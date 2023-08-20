package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jeffprestes/sjrpc/database"
	"github.com/jeffprestes/sjrpc/handler"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	whereAmI, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(whereAmI, "database", "data")
	database.DB, err = database.NewBadgerDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.DB.Close()

	webserver := echo.New()
	webserver.Use(middleware.Logger())
	webserver.Use(middleware.Recover())

	webserver.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "Hello, This is Save JSON-RPC")
	})

	webserver.POST("/", handler.PostHandler)

	httpPort := "8434"

	webserver.Logger.Fatal(webserver.Start(":" + httpPort))
}
