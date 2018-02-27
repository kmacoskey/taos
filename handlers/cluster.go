package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/services"
	log "github.com/sirupsen/logrus"
)

type clusterService interface {
	GetCluster(rc app.RequestContext, id string) (*models.Cluster, error)
	GetClusters(rc app.RequestContext) ([]models.Cluster, error)
	CreateCluster(rc app.RequestContext) (*models.Cluster, error)
	DeleteCluster(rc app.RequestContext, id string) (*models.Cluster, error)
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
	h := NewClusterHandler(services.NewClusterService(daos.NewClusterDao(), db))

	router.Handle("/cluster/{id}", app.Adapt(
		router,
		h.GetCluster(),
		app.WithRequestContext(),
	)).Methods("GET")

	router.Handle("/clusters", app.Adapt(
		router,
		h.GetClusters(),
		app.WithRequestContext(),
	)).Methods("GET")

	router.Handle("/cluster", app.Adapt(
		router,
		h.CreateCluster(),
		app.WithRequestContext(),
	)).Methods("PUT")

	router.Handle("/cluster/{id}", app.Adapt(
		router,
		h.DeleteCluster(),
		app.WithRequestContext(),
	)).Methods("DELETE")
}

// Request provisioning of a new Cluster
func (ch *ClusterHandler) CreateCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"context": "cluster_handler",
				"event":   "create_cluster",
			})

			rc := app.GetRequestContext(r)

			body, _ := ioutil.ReadAll(r.Body)
			rc.SetTerraformConfig(body)

			cluster, err := ch.cs.CreateCluster(rc)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Error("could not create cluster")
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

// Retrieve a single Cluster for a given id
func (ch *ClusterHandler) GetCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"context": "cluster_handler",
				"event":   "getcluster",
			})

			vars := mux.Vars(r)
			rc := app.GetRequestContext(r)

			id := vars["id"]

			cluster, err := ch.cs.GetCluster(rc, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				logger.Debug("could not retrieve cluster for given id in request")
				return
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

// Delete a Cluster for a given id
func (ch *ClusterHandler) DeleteCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"context": "cluster_handler",
				"event":   "deletecluster",
			})

			vars := mux.Vars(r)
			rc := app.GetRequestContext(r)

			id := vars["id"]

			cluster, err := ch.cs.DeleteCluster(rc, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				logger.Debug("could not retrieve cluster for given id in request")
			}

			if cluster != nil {
				js, err := json.Marshal(cluster)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					logger.Panic("failed to marshal cluster data for response")
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
			}
		})
	}
}
