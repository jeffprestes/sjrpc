package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/jeffprestes/sjrpc/database"
	"github.com/labstack/echo/v4"
)

func DbCleanHandler(echoCtx echo.Context) error {
	var err error
	err = database.DB.Close()
	if err != nil {
		return err
	}
	whereAmI, err := os.Getwd()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(whereAmI, "database", "data")
	err = os.RemoveAll(dbPath)
	if err != nil {
		return err
	}
	database.DB, err = database.NewBadgerDB(dbPath)
	if err != nil {
		return err
	}
	echoCtx.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	return echoCtx.String(http.StatusOK, "{'status':'ok'}")
}
