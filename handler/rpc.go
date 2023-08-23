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
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/dgraph-io/badger/v4"
	"github.com/jeffprestes/sjrpc/database"
	"github.com/jeffprestes/sjrpc/localcache"
	"github.com/jeffprestes/sjrpc/model"
	"github.com/labstack/echo/v4"
)

var (
	RpcUrl              string
	debug               bool
	userSelectedChainId *int
)

func init() {
	if len(strings.TrimSpace(os.Getenv("SJRPC_URL"))) < 5 {
		log.Fatalln("no SJRPC_URL server set in environment variable")
	}
	RpcUrl = os.Getenv("SJRPC_URL")
}

func PostHandler(echoCtx echo.Context) error {
	if echoCtx.QueryParam("debug") == "1" ||
		strings.ToLower(echoCtx.QueryParam("debug")) == "true" {
		debug = true
	}

	var err error
	echoCtx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	contentType := echoCtx.Request().Header["Content-Type"][0]
	if len(contentType) < 10 || strings.ToLower(contentType) != "application/json" {
		log.Print("\n")
		log.Println("********************************************")
		log.Println("What is echoCtx.Request().Header[Content-Type] ?", echoCtx.Request().Header["Content-Type"][0])
		log.Println("********************************************")
		log.Print("\n")
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
		requestHash := request.Base64Hash()

		if cacheUsed && debug {
			log.Print("\n\n")
			log.Println(" +++ request: ", requestHash)
			log.Printf("%+v", request)
			log.Print("\n\n")
		}

		if request.IsCacheable() {
			resp, err = database.DB.Get(database.RequestNamespace, request.Hash())
			if err == badger.ErrKeyNotFound {
				resp, err = PerformRemoteCall(echoCtx, &request)
				if err != nil {
					return err
				}
				database.DB.Insert(database.RequestNamespace, request.Hash(), []byte(resp))
				cacheUsed = false
			} else if err != nil {
				return err
			}
		} else if request.IsAfterFinalCacheable() {
			resp, err = database.DB.Get(database.RequestNamespace, request.Hash())
			if err == badger.ErrKeyNotFound {
				resp, err = PerformRemoteCall(echoCtx, &request)
				if err != nil {
					return err
				}
				if request.IsResultFinal(resp) {
					database.DB.Insert(database.RequestNamespace, request.Hash(), []byte(resp))
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
			tmpObj, ok := localcache.TimelyRequests.Load(request.Base64Hash())
			if !ok {
				respObj, err = PerformRemoteCallForTimelyEndpoints(echoCtx, &request)
				if err != nil {
					return err
				}
				localcache.TimelyRequests.Store(request.Base64Hash(), respObj)
				cacheUsed = false
			} else {
				log.Println("Request base64hash: ", request.Base64Hash())
				respObj = tmpObj.(model.EphemeralRequest)
				if !respObj.IsStillValid() {
					respObj, err = PerformRemoteCallForTimelyEndpoints(echoCtx, &request)
					if err != nil {
						return err
					}
					_, swapped := localcache.TimelyRequests.Swap(request.Base64Hash(), respObj)
					if swapped {
						if debug {
							log.Println(request.Base64Hash(), " has been updated")
						}
					}
					cacheUsed = false
				}
			}
			resp = respObj.Response
		} else {
			resp, err = PerformRemoteCall(echoCtx, &request)
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

func PerformRemoteCall(echoCtx echo.Context, request *model.RPCRequest) (resp string, err error) {
	tmpResp := new(bytes.Buffer)
	err = requests.URL(RpcUrl).BodyJSON(request).ContentType("application/json").ToBytesBuffer(tmpResp).Fetch(echoCtx.Request().Context())
	if err != nil {
		return
	}
	resp = tmpResp.String()
	return
}

func GetLatestBlockInfo(echoCtx echo.Context) (resp model.BlockByNumberResponse, err error) {
	var request model.RPCRequest
	request.JsonRpcVersion = "2.0"
	request.Method = "eth_blockNumber"
	request.ID = 1

	blockNumberResp := new(model.BlockNumberResponse)
	err = requests.URL(RpcUrl).BodyJSON(request).ContentType("application/json").ToJSON(blockNumberResp).Fetch(echoCtx.Request().Context())
	if err != nil {
		return
	}

	request.JsonRpcVersion = "2.0"
	request.Method = "eth_getBlockByNumber"
	request.ID = 1
	request.Params = append(request.Params, blockNumberResp.Result)
	request.Params = append(request.Params, true)
	blockByNumberResp := new(model.BlockByNumberResponse)
	err = requests.URL(RpcUrl).BodyJSON(request).ContentType("application/json").ToJSON(blockByNumberResp).Fetch(echoCtx.Request().Context())
	if err != nil {
		return
	}
	resp = *blockByNumberResp
	return
}

func GetLatestBlockTimestamp(echoCtx echo.Context) (timestamp uint64, err error) {
	block, err := GetLatestBlockInfo(echoCtx)
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

func PerformRemoteCallForTimelyEndpoints(echoCtx echo.Context, request *model.RPCRequest) (respObj model.EphemeralRequest, err error) {
	var lastBlock model.BlockByNumberResponse
	lastBlock, err = GetLatestBlockInfo(echoCtx)
	if err != nil {
		return
	}
	respObj.BlockNumber = ConvertStrRespToUInt64(lastBlock.Result.Number)
	if respObj.BlockNumber < 1000 {
		err = fmt.Errorf("could not convert block number to int: %s", lastBlock.Result.Number)
		return
	}
	respObj.When = ConvertStrRespToInt64(lastBlock.Result.Timestamp)
	if respObj.When < 1000 {
		err = fmt.Errorf("could not convert block number to timestamp: %s", lastBlock.Result.Timestamp)
		return
	}
	newResp, err := PerformRemoteCall(echoCtx, request)
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
	tmpStr, _ := strings.CutPrefix(strNumber, "0x")
	tmpInt, ok := new(big.Int).SetString(tmpStr, 16)
	if ok {
		value = tmpInt.Uint64()
	}
	return
}
