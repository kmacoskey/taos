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

// ClusterList represents a list of returned Clusters
type ClusterList struct {
	TotalCount int         `json:"total_count"`
	Clusters   interface{} `json:"clusters"`
}

// Retrieve a single Cluster for a given id
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

// Retrieve a ClusterList of all Clusters
func GetClusters() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic": "taos",
				"event": "cluster_handler",
			})

			rc := app.GetRequestContext(r)

			clusters, err := services.GetClusters(rc)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Error("could not retrieve clusters")
			}

			var clusterlist = ClusterList{
				TotalCount: len(clusters),
				Clusters:   clusters,
			}

			js, err := json.Marshal(clusterlist)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Panic("failed to marshal cluster data for response")
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		})
	}
}
