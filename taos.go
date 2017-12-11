package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/kmacoskey/taos/middleware"
	"net/http"
)

var config serverConfig

func main() {
	// Server configuration
	if err := LoadServerConfig(&config); err != nil {
		panic(fmt.Errorf("Invalid application configuration: %s", err))
	}

	// Logging
	if err := InitLogger(); err != nil {
		panic(fmt.Errorf("Logging Initialization Failed: %s", err))
	}

	// Database Connection
	db, err := DatabaseConnect()
	if err != nil {
		panic(fmt.Errorf("Connection to Database Failed: %s", err))
	}
	defer db.Close()

	// Wire up the routing
	router := mux.NewRouter()

	router.Handle("/", middleware.Adapt(router,
		IndexHandler(),
		middleware.Logging(),
	))

	// Start the server
	http.ListenAndServe(":8080", router)
}

func IndexHandler() middleware.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
		})
	}
}
