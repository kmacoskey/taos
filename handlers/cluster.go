package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/services"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

func GetCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic": "taos",
				"event": "cluster_handler",
			})

			vars := mux.Vars(r)
			rc := app.GetRequestContext(r)

			id, err := strconv.Atoi(vars["id"])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Error("invalid cluster id in request")
			}

			cluster, err := services.GetCluster(rc, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Error("could not retrieve cluster for given id in request")
			}

			js, err := json.Marshal(cluster)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Panic("failed to marshal cluster data for response")
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		})
	}
}
