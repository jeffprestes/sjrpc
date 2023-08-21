package handler

import (
	"bytes"
	"fmt"
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
	"github.com/labstack/echo"
)

var (
	RpcUrl string
	debug  bool
)

func init() {
	if len(strings.TrimSpace(os.Getenv("SJRPC_URL"))) < 5 {
		log.Fatalln("no SJRPC_URL server set in environment variable")
	}
	RpcUrl = os.Getenv("SJRPC_URL")
	debug = true
}

func PostHandler(echoCtx echo.Context) error {
	var err error
	echoCtx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	if echoCtx.Request().ContentLength <= 237 {
		return echoCtx.String(http.StatusOK, "")
	}

	request := new(model.RPCRequest)
	err = echoCtx.Bind(request)
	if err != nil {
		log.Printf("request: %+v", echoCtx.Request())
		return err
	}
	var resp string
	cacheUsed := true
	requestHash := request.Base64Hash()

	if cacheUsed && debug {
		log.Print("\n\n")
		log.Println(" +++ request: ", requestHash)
		log.Print("\n\n")
	}

	if request.IsCacheable() {
		resp, err = database.DB.Get(database.RequestNamespace, request.Hash())
		if err == badger.ErrKeyNotFound {
			resp, err = PerformRemoteCall(echoCtx, request)
			if err != nil {
				return err
			}
			database.DB.Insert(database.RequestNamespace, request.Hash(), []byte(resp))
			cacheUsed = false
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
			respObj, err = PerformRemoteCallForTimelyEndpoints(echoCtx, request)
			if err != nil {
				return err
			}
			localcache.TimelyRequests.Store(request.Base64Hash(), respObj)
			cacheUsed = false
		} else {
			log.Println("Request base64hash: ", request.Base64Hash())
			respObj = tmpObj.(model.EphemeralRequest)
			if !respObj.IsStillValid() {
				respObj, err = PerformRemoteCallForTimelyEndpoints(echoCtx, request)
				if err != nil {
					return err
				}
				_, swapped := localcache.TimelyRequests.Swap(request.Base64Hash(), respObj)
				if swapped {
					log.Println(request.Base64Hash(), " has been updated")
				}
				cacheUsed = false
			}
			// now := time.Now().UTC().Unix()
			// max := respObj.When + 12
			// log.Println("now: ", now, "-", "max:", max, " - valid?", now <= max)
		}
		resp = respObj.Response
	} else {
		resp, err = PerformRemoteCall(echoCtx, request)
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
	echoCtx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	return echoCtx.String(http.StatusOK, resp)
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
