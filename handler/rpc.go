package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/dgraph-io/badger/v4"
	"github.com/jeffprestes/sjrpc/database"
	"github.com/jeffprestes/sjrpc/localcache"
	"github.com/jeffprestes/sjrpc/model"
	"github.com/labstack/echo/v4"
)

func PostHandler(echoCtx echo.Context) error {
	var err error

	// Function params
	debug, userSelectedChainId, rpcUrl := CheckParams(echoCtx)
	if len(rpcUrl) < 5 {
		err = fmt.Errorf("no SJRPC_URL server set in environment variable or query string")
		return err
	}

	echoCtx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	contentType := echoCtx.Request().Header["Content-Type"][0]
	if len(contentType) < 10 || strings.ToLower(contentType) != "application/json" {
		if debug {
			log.Print("\n")
			log.Println("********************************************")
			log.Println("What is echoCtx.Request().Header[Content-Type] ?", echoCtx.Request().Header["Content-Type"][0])
			log.Println("********************************************")
			log.Print("\n")
		}
		err = fmt.Errorf("invalid content-type header: %s", contentType)
		echoCtx.JSON(http.StatusBadRequest, err)
		return err
	}

	body, errReadBytes := io.ReadAll(echoCtx.Request().Body)
	if errReadBytes != nil {
		log.Printf("error reading request bytes: %s\n", errReadBytes.Error())
		return errReadBytes
	}
	var requests []model.RPCRequest
	var request model.RPCRequest
	errDecode := json.Unmarshal(body, &request)
	if errDecode != nil {
		errDecode = json.Unmarshal(body, &requests)
		if errDecode != nil {
			log.Printf("decoding request error: %s\n", errDecode.Error())
			return errDecode
		}
		if debug {
			log.Println("Number of requests: ", len(requests))
		}
	} else {
		requests = append(requests, request)
	}

	var respFinal strings.Builder
	var resp string
	cacheUsed := true

	for i := 0; i < len(requests); i++ {
		request = requests[i]
		requestHash := request.Base64Hash(userSelectedChainId)

		if cacheUsed && debug {
			log.Print("\n\n")
			log.Println(" +++ request: ", requestHash)
			log.Printf("%+v", request)
			log.Print("\n\n")
		}

		if request.IsCacheable() {
			resp, err = database.DB.Get(database.RequestNamespace, request.Hash(userSelectedChainId))
			if err == badger.ErrKeyNotFound {
				resp, err = PerformRemoteCall(echoCtx, &request, rpcUrl)
				if err != nil {
					return err
				}
				database.DB.Insert(database.RequestNamespace, request.Hash(userSelectedChainId), []byte(resp))
				cacheUsed = false
			} else if err != nil {
				return err
			}
		} else if request.IsAfterFinalCacheable() {
			resp, err = database.DB.Get(database.RequestNamespace, request.Hash(userSelectedChainId))
			if err == badger.ErrKeyNotFound {
				resp, err = PerformRemoteCall(echoCtx, &request, rpcUrl)
				if err != nil {
					return err
				}
				if request.IsResultFinal(resp) {
					database.DB.Insert(database.RequestNamespace, request.Hash(userSelectedChainId), []byte(resp))
					cacheUsed = false
				}
			} else if err != nil {
				return err
			}
		} else if request.IsEnvCacheable() {
			if strings.ToLower(request.Method) == "eth_accounts" {
				respJson := model.AccountResponse{}
				respJson.ID = request.ID
				respJson.Jsonrpc = request.JsonRpcVersion
				respJson.Result = append(respJson.Result, os.Getenv("ETH_FROM"))
				resp = respJson.ToString()
			}
		} else if request.IsTimelyCacheable() {
			var respObj model.EphemeralRequest
			tmpObj, ok := localcache.TimelyRequests.Load(request.Base64Hash(userSelectedChainId))
			if !ok {
				respObj, err = PerformRemoteCallForTimelyEndpoints(echoCtx, &request, rpcUrl)
				if err != nil {
					if debug {
						log.Printf("request.IsTimelyCacheable - PerformRemoteCallForTimelyEndpoints: %s\n", err.Error())
					}
					return err
				}
				localcache.TimelyRequests.Store(request.Base64Hash(userSelectedChainId), respObj)
				cacheUsed = false
			} else {
				if debug {
					log.Println("Request base64hash: ", request.Base64Hash(userSelectedChainId))
				}
				respObj = tmpObj.(model.EphemeralRequest)
				if !respObj.IsStillValid() {
					respObj, err = PerformRemoteCallForTimelyEndpoints(echoCtx, &request, rpcUrl)
					if err != nil {
						return err
					}
					_, swapped := localcache.TimelyRequests.Swap(request.Base64Hash(userSelectedChainId), respObj)
					if swapped {
						if debug {
							log.Println(request.Base64Hash(userSelectedChainId), " has been updated")
						}
					}
					cacheUsed = false
				}
			}
			resp = respObj.Response
		} else {
			resp, err = PerformRemoteCall(echoCtx, &request, rpcUrl)
			if err != nil {
				return err
			}
			cacheUsed = false
		}

		if cacheUsed && debug {
			log.Print("\n\n")
			log.Println(" *** cache was used for the request: ", requestHash)
			log.Print("\n\n")
		}

		if len(requests) > 1 {
			resp = RestoreOriginalId(&request, resp)
			if i > 0 && i < len(requests) {
				resp = "," + resp
			} else if i == 0 {
				resp = "[" + resp
			}
		}

		_, err = respFinal.WriteString(resp)
		if err != nil {
			log.Println("error writing response into response buffer: ", err.Error())
			return err
		}
		if debug {
			log.Println("resp added: ", resp, " - respFinal: ", respFinal.String())
		}
	}
	if len(requests) > 1 {
		respFinal.WriteString("]")
	}

	if debug {
		log.Print("Response:\n", respFinal.String(), "\n\n")
	}

	return echoCtx.String(http.StatusOK, respFinal.String())
}

func PerformRemoteCall(echoCtx echo.Context, request *model.RPCRequest, rpcUrl string) (resp string, err error) {
	tmpResp := new(bytes.Buffer)
	err = requests.URL(rpcUrl).BodyJSON(request).ContentType("application/json").ToBytesBuffer(tmpResp).Fetch(echoCtx.Request().Context())
	if err != nil {
		return
	}
	resp = tmpResp.String()
	return
}

func GetLatestBlockInfo(echoCtx echo.Context, rpcUrl string) (resp model.BlockByNumberResponse, err error) {
	var request model.RPCRequest
	request.JsonRpcVersion = "2.0"
	request.Method = "eth_blockNumber"
	request.ID = 1

	blockNumberResp := new(model.BlockNumberResponse)
	err = requests.URL(rpcUrl).BodyJSON(request).ContentType("application/json").ToJSON(blockNumberResp).Fetch(echoCtx.Request().Context())
	if err != nil {
		return
	}

	request.JsonRpcVersion = "2.0"
	request.Method = "eth_getBlockByNumber"
	request.ID = 1
	request.Params = append(request.Params, blockNumberResp.Result)
	request.Params = append(request.Params, true)
	blockByNumberResp := new(model.BlockByNumberResponse)
	err = requests.URL(rpcUrl).BodyJSON(request).ContentType("application/json").ToJSON(blockByNumberResp).Fetch(echoCtx.Request().Context())
	if err != nil {
		return
	}
	resp = *blockByNumberResp
	return
}

func GetLatestBlockTimestamp(echoCtx echo.Context, rpcUrl string) (timestamp uint64, err error) {
	block, err := GetLatestBlockInfo(echoCtx, rpcUrl)
	if err != nil {
		return
	}
	tmpInt, ok := big.NewInt(0).SetString(block.Result.Timestamp, 16)
	if !ok {
		err = fmt.Errorf("block timestamp %s conversion error", block.Result.Timestamp)
		return
	}
	timestamp = tmpInt.Uint64()
	return
}

func PerformRemoteCallForTimelyEndpoints(echoCtx echo.Context, request *model.RPCRequest, rpcUrl string) (respObj model.EphemeralRequest, err error) {
	var lastBlock model.BlockByNumberResponse
	lastBlock, err = GetLatestBlockInfo(echoCtx, rpcUrl)
	if err != nil {
		return
	}
	respObj.BlockNumber = ConvertStrRespToUInt64(lastBlock.Result.Number)
	if respObj.BlockNumber < 1000 {
		if !strings.Contains(rpcUrl, "localhost") && !strings.Contains(rpcUrl, "127.0.0.1") {
			err = fmt.Errorf("could not convert block number to int: %s", lastBlock.Result.Number)
			return
		}
	}
	respObj.When = ConvertStrRespToInt64(lastBlock.Result.Timestamp)
	if respObj.When < 1000 {
		err = fmt.Errorf("could not convert block number to timestamp: %s", lastBlock.Result.Timestamp)
		return
	}
	newResp, err := PerformRemoteCall(echoCtx, request, rpcUrl)
	if err != nil {
		return
	}
	respObj.Request = *request
	respObj.Response = newResp
	return
}

func ConvertStrRespToInt64(strNumber string) (value int64) {
	tmpStr, _ := strings.CutPrefix(strNumber, "0x")
	tmpInt, ok := new(big.Int).SetString(tmpStr, 16)
	if ok {
		value = tmpInt.Int64()
	}
	return
}

func ConvertStrRespToUInt64(strNumber string) (value uint64) {
	tmpStr, found := strings.CutPrefix(strNumber, "0x")
	if !found {
		value = 0
	}
	tmpStr = strings.TrimSpace(tmpStr)
	tmpInt, ok := new(big.Int).SetString(tmpStr, 16)
	if ok {
		value = tmpInt.Uint64()
	}
	// log.Println("ConvertStrRespToUInt64 - strNumber: ", strNumber, " - found: ", found, " - tmpStr: ", tmpStr, " - ok: ", ok, " - tmpInt: ", tmpInt, " - value: ", value)
	return
}

func CheckParams(echoCtx echo.Context) (debug bool, chainId *int, rpcUrl string) {
	if echoCtx.QueryParam("debug") == "1" ||
		strings.ToLower(echoCtx.QueryParam("debug")) == "true" {
		debug = true
	}

	if echoCtx.Request().URL.Query().Has("rpcurl") ||
		echoCtx.Request().URL.Query().Has("rpc_url") ||
		echoCtx.Request().URL.Query().Has("rpcUrl") ||
		echoCtx.Request().URL.Query().Has("RPCURL") {
		if len(echoCtx.QueryParam("rpcurl")) > 0 {
			rpcUrl = echoCtx.QueryParam("rpcurl")
		} else if len(echoCtx.QueryParam("rpc_url")) > 0 {
			rpcUrl = echoCtx.QueryParam("rpc_url")
		} else if len(echoCtx.QueryParam("RPCURL")) > 0 {
			rpcUrl = echoCtx.QueryParam("RPCURL")
		} else if len(echoCtx.QueryParam("rpcUrl")) > 0 {
			rpcUrl = echoCtx.QueryParam("rpcUrl")
		}
	} else {
		rpcUrl = os.Getenv("SJRPC_URL")
	}

	if echoCtx.Request().URL.Query().Has("chainId") ||
		echoCtx.Request().URL.Query().Has("chainid") ||
		echoCtx.Request().URL.Query().Has("CHAINID") {
		var str string
		if len(echoCtx.QueryParam("chainId")) > 0 {
			str = echoCtx.QueryParam("chainId")
		} else if len(echoCtx.QueryParam("chainid")) > 0 {
			str = echoCtx.QueryParam("chainid")
		} else if len(echoCtx.QueryParam("CHAINID")) > 0 {
			str = echoCtx.QueryParam("CHAINID")
		}

		if len(str) > 0 {
			var intTmp int
			intTmp, err := strconv.Atoi(str)
			if err == nil {
				chainId = &intTmp
			} else {
				chainId = nil
			}
		}
	}
	return
}

func RestoreOriginalId(request *model.RPCRequest, resp string) (newResp string) {
	exp := "\"id\":[0-9]+"
	re, err := regexp.CompilePOSIX(exp)
	if err != nil {
		return
	}
	tmpId := "\"id\":" + strconv.Itoa(request.ID)
	newResp = re.ReplaceAllString(resp, tmpId)
	return
}
