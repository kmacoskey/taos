package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/handlers"
	"net/http"
)

var config app.ServerConfig

func main() {
	// Server configuration
	if err := app.LoadServerConfig(&config, "."); err != nil {
		panic(fmt.Errorf("Invalid application configuration: %s", err))
	}

	// Logging
	if err := app.InitLogger(config.Logging); err != nil {
		panic(fmt.Errorf("Logging Initialization Failed: %s", err))
	}

	// Database Connection
	db, err := app.DatabaseConnect(config.ConnStr)
	if err != nil {
		panic(fmt.Errorf("Connection to Database Failed: %s", err))
	}
	defer db.Close()

	// Routing
	router := mux.NewRouter()

	handlers.ServeClusterResources(router, db)

	// Start the server
	http.ListenAndServe(":8080", router)
}
