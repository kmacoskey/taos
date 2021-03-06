package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/handlers"
	"github.com/kmacoskey/taos/reaper"
	"github.com/kmacoskey/taos/services"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := app.LoadServerConfig(&app.GlobalServerConfig, "."); err != nil {
		panic(fmt.Errorf("Invalid application configuration: %s", err))
	}

	if err := app.InitLogger(app.GlobalServerConfig.Logging); err != nil {
		panic(fmt.Errorf("Logging Initialization Failed: %s", err))
	}

	db, err := app.DatabaseConnect(app.GlobalServerConfig.ConnStr)
	if err != nil {
		panic(fmt.Errorf("Connection to Database Failed: %s", err))
	}

	defer db.Close()

	router := mux.NewRouter()
	handlers.ServeClusterResources(router, db)

	reaper, _ := reaper.NewClusterReaper(app.GlobalServerConfig.ReapInterval, services.NewClusterService(daos.NewClusterDao(), db), db)
	reaper.StartReaping()

	_ = StartHttpServer(router)
	// Process control is expected to be handled from the environment
	//  therefore there is no reason to use the returned server to call
	//  Shutdown()
}

func StartHttpServer(router *mux.Router) *http.Server {
	logger := log.WithFields(log.Fields{"package": "taos", "event": "start_http", "request": ""})

	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", app.GlobalServerConfig.ServerPort),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if app.GlobalServerConfig.BackgroundForTesting {
		go func() {
			if err := server.ListenAndServe(); err != nil {
				// This is most likely an intentional close
				logger.Info(err)
			}
		}()

	} else {
		server.ListenAndServe()
	}

	// return reference so caller can call Shutdown() if desired
	return server
}
