package handler

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/dgraph-io/badger/v4"
	"github.com/jeffprestes/sjrpc/database"
	"github.com/jeffprestes/sjrpc/model"
	"github.com/labstack/echo"
)

var RpcUrl string

func init() {
	if len(strings.TrimSpace(os.Getenv("SJRPC_URL"))) < 5 {
		log.Fatalln("no SJRPC_URL server set in environment variable")
	}
	RpcUrl = os.Getenv("SJRPC_URL")
}

func PostHandler(echoCtx echo.Context) error {
	var err error

	request := new(model.RPCRequest)

	err = echoCtx.Bind(request)
	if err != nil {
		return err
	}

	var resp string

	if request.IsCacheable() {
		resp, err = database.DB.Get(database.RequestNamespace, request.Hash())
		if err == badger.ErrKeyNotFound {
			tmpResp := new(bytes.Buffer)
			err = requests.URL(RpcUrl).BodyJSON(request).ContentType("application/json").ToBytesBuffer(tmpResp).Fetch(echoCtx.Request().Context())
			if err != nil {
				return err
			}
			resp = tmpResp.String()
			database.DB.Insert(database.RequestNamespace, request.Hash(), []byte(resp))
		} else if err != nil {
			return err
		}
	}
	echoCtx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	return echoCtx.String(http.StatusOK, resp)
}
