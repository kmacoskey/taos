package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/middleware"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/services"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type clusterService interface {
	GetCluster(rc app.RequestContext, id int) (*models.Cluster, error)
	GetClusters(rc app.RequestContext) ([]models.Cluster, error)
}

type ClusterHandler struct {
	cs clusterService
}

func NewClusterHandler(cs clusterService) *ClusterHandler {
	return &ClusterHandler{cs}
}

// ClusterList represents a list of returned Clusters
type ClusterList struct {
	TotalCount int         `json:"total_count"`
	Clusters   interface{} `json:"clusters"`
}

func ServeClusterResources(router *mux.Router, db *sqlx.DB) {
	h := NewClusterHandler(services.NewClusterService(daos.NewClusterDao()))

	router.Handle("/cluster/{id}", app.Adapt(
		router,
		h.GetCluster(),
		middleware.Transactional(db),
		app.WithRequestContext(),
	)).Methods("GET")

	router.Handle("/clusters", app.Adapt(
		router,
		h.GetClusters(),
		middleware.Transactional(db),
		app.WithRequestContext(),
	)).Methods("GET")

}

// Retrieve a single Cluster for a given id
func (ch *ClusterHandler) GetCluster() app.Adapter {
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

			cluster, err := ch.cs.GetCluster(rc, id)
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
func (ch *ClusterHandler) GetClusters() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic": "taos",
				"event": "cluster_handler",
			})

			rc := app.GetRequestContext(r)

			clusters, err := ch.cs.GetClusters(rc)
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
