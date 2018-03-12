package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
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

type RequestResponse struct {
	RequestId string       `json:"request_id"`
	Status    string       `json:"status"`
	Data      ResponseData `json:"data"`
}

type ResponseData struct {
	Type       string `json:"type"`
	Attributes interface{}
}

type ResponseAttributes interface {
}

type ClusterResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ErrorResponse struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
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

			request_id := uuid.New()

			body, _ := ioutil.ReadAll(r.Body)
			rc.SetTerraformConfig(body)

			cluster, err := ch.cs.CreateCluster(rc)

			rd := ResponseData{}

			if len(body) <= 0 {
				er := ErrorResponse{
					Title:  "incorrect_request_paramaters",
					Detail: "Missing required terraform configuration for create cluster request",
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusBadRequest)

			} else if cluster != nil {

				cr := ClusterResponse{
					Id:     cluster.Id,
					Name:   cluster.Name,
					Status: cluster.Status,
				}

				rd = ResponseData{
					Type:       "cluster",
					Attributes: cr,
				}

				w.WriteHeader(http.StatusAccepted)

			} else {

				er := ErrorResponse{
					Title:  "create_cluster_error",
					Detail: "Failed to create cluster",
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusInternalServerError)

			}

			rr := RequestResponse{
				RequestId: request_id.String(),
				Status:    "foo",
				Data:      rd,
			}

			js, err := json.Marshal(rr)
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
			request_id := uuid.New()

			id := vars["id"]

			cluster, err := ch.cs.GetCluster(rc, id)
			if err != nil {
				logger.Debug("could not retrieve cluster for given id in request")
			}

			rd := ResponseData{}

			if len(id) <= 0 {
				er := ErrorResponse{
					Title:  "incorrect_request_paramaters",
					Detail: "Missing required cluster id",
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusBadRequest)

			} else if cluster != nil {

				cr := ClusterResponse{
					Id:     cluster.Id,
					Name:   cluster.Name,
					Status: cluster.Status,
				}

				rd = ResponseData{
					Type:       "cluster",
					Attributes: cr,
				}

				w.WriteHeader(http.StatusOK)

			} else if err == nil && cluster == nil {
				er := ErrorResponse{
					Title:  "get_cluster_error",
					Detail: "Cluster does not exist",
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusNotFound)

			} else if err != nil {

				er := ErrorResponse{
					Title:  "get_cluster_error",
					Detail: err.Error(),
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusInternalServerError)

			}

			rr := RequestResponse{
				RequestId: request_id.String(),
				Status:    "foo",
				Data:      rd,
			}

			js, err := json.Marshal(rr)
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
			request_id := uuid.New()
			rd := ResponseData{}

			clusters, err := ch.cs.GetClusters(rc)

			if clusters != nil {

				cluster_list := []ClusterResponse{}

				for _, cluster := range clusters {
					cr := ClusterResponse{
						Id:     cluster.Id,
						Name:   cluster.Name,
						Status: cluster.Status,
					}

					cluster_list = append(cluster_list, cr)
				}

				rd = ResponseData{
					Type:       "clusters",
					Attributes: cluster_list,
				}

				w.WriteHeader(http.StatusOK)

			} else if err != nil {

				er := ErrorResponse{
					Title:  "get_clusters_error",
					Detail: err.Error(),
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusInternalServerError)

			}

			rr := RequestResponse{
				RequestId: request_id.String(),
				Status:    "foo",
				Data:      rd,
			}

			js, err := json.Marshal(rr)
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
			rd := ResponseData{}
			request_id := uuid.New()

			id := vars["id"]

			cluster, err := ch.cs.DeleteCluster(rc, id)

			if len(id) <= 0 {
				er := ErrorResponse{
					Title:  "incorrect_request_paramaters",
					Detail: "Missing required cluster id",
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusBadRequest)

			} else if cluster != nil {

				cr := ClusterResponse{
					Id:     cluster.Id,
					Name:   cluster.Name,
					Status: cluster.Status,
				}

				rd = ResponseData{
					Type:       "cluster",
					Attributes: cr,
				}

				w.WriteHeader(http.StatusAccepted)

			} else if err == nil && cluster == nil {
				er := ErrorResponse{
					Title:  "delete_cluster_error",
					Detail: "Cluster does not exist",
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusNotFound)

			} else if err != nil {

				er := ErrorResponse{
					Title:  "delete_cluster_error",
					Detail: err.Error(),
				}

				rd = ResponseData{
					Type:       "error",
					Attributes: er,
				}

				w.WriteHeader(http.StatusInternalServerError)

			}

			rr := RequestResponse{
				RequestId: request_id.String(),
				Status:    "foo",
				Data:      rd,
			}

			js, err := json.Marshal(rr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Panic("failed to marshal cluster data for response")
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		})
	}
}
