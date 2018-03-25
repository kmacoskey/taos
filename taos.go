package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/handlers"
)

// var ServerConfig app.ServerConfig

func main() {
	// Server configuration
	if err := app.LoadServerConfig(&app.GlobalServerConfig, "."); err != nil {
		panic(fmt.Errorf("Invalid application configuration: %s", err))
	}

	// Logging
	if err := app.InitLogger(app.GlobalServerConfig.Logging); err != nil {
		panic(fmt.Errorf("Logging Initialization Failed: %s", err))
	}

	// Database Connection
	db, err := app.DatabaseConnect(app.GlobalServerConfig.ConnStr)
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
