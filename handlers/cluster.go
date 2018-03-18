package handlers

import (
	"encoding/json"
	"fmt"
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

type RequestResponse struct {
	RequestId string       `json:"request_id"`
	Status    string       `json:"status"`
	Data      ResponseData `json:"data"`
}

type ResponseData struct {
	Type       string `json:"type"`
	Attributes interface{}
}

type ResponseAttributes interface{}

type ClusterResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ErrorResponse struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func NewClusterHandler(cs clusterService) *ClusterHandler {
	return &ClusterHandler{cs}
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

func newErrorResponse(response *ErrorResponse, request_id string) *RequestResponse {
	response_data := ResponseData{
		Type:       "error",
		Attributes: response,
	}

	request_response := RequestResponse{
		RequestId: request_id,
		Data:      response_data,
	}

	return &request_response
}

// Request provisioning of a new Cluster
func (ch *ClusterHandler) CreateCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "create_cluster",
				"request": context.RequestId(),
			})

			elogger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "create_cluster_error",
				"request": context.RequestId(),
			})

			logger.Info("request new cluster")

			body, _ := ioutil.ReadAll(r.Body)

			// Will not continue if missing terraform configuration in request
			if len(body) <= 0 {
				response := ErrorResponse{
					Title:  "create_cluster_error",
					Detail: "Missing required terraform configuration for create cluster request",
				}

				elogger.Error("missing terraform config")
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusBadRequest)
				return
			}

			logger.Debug(body)

			context.SetTerraformConfig(body)

			cluster, err := ch.cs.CreateCluster(context)
			// Internal error in any layer below handler
			if err != nil {
				response := ErrorResponse{
					Title:  "create_cluster_error",
					Detail: err.Error(),
				}

				elogger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			// No expectation for the situation that
			// 	err == nil && cluster == nil
			// Currently if a cluster is not returned, then something went wrong
			// Eventually this may capture the situation where resources are not available

			logger.Info("cluster created")
			logger.Debug(cluster)
			respondWithJson(w, newClusterResponse(cluster, context.RequestId()), http.StatusAccepted)
		})
	}
}

// Retrieve a single Cluster for a given id
func (ch *ClusterHandler) GetCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "get_cluster",
				"request": context.RequestId(),
			})

			elogger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "get_cluster_error",
				"request": context.RequestId(),
			})

			vars := mux.Vars(r)
			id := vars["id"]

			logger.Info(fmt.Sprintf("get cluster '%s'", id))

			// Will not continue if missing id in request
			if len(id) <= 0 {
				response := ErrorResponse{
					Title:  "get_cluster_error",
					Detail: "Missing required cluster id",
				}

				elogger.Error("missing id")
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusBadRequest)
				return
			}

			cluster, err := ch.cs.GetCluster(context, id)
			// Internal error in any layer below handler
			if err != nil {
				response := ErrorResponse{
					Title:  "get_cluster_error",
					Detail: err.Error(),
				}

				elogger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			// Cluster does not exist
			if cluster == nil {
				response := ErrorResponse{
					Title:  "get_cluster_error",
					Detail: "cluster not found",
				}

				elogger.Error("cluster not found")
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusNotFound)
				return
			}

			logger.Info("returning cluster")
			logger.Debug(cluster)
			respondWithJson(w, newClusterResponse(cluster, context.RequestId()), http.StatusOK)
		})
	}
}

// Retrieve a ClusterList of all Clusters
func (ch *ClusterHandler) GetClusters() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "get_clusters",
				"request": context.RequestId(),
			})

			elogger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "get_clusters_error",
				"request": context.RequestId(),
			})

			clusters, err := ch.cs.GetClusters(context)
			// Internal error in any layer below handler
			if err != nil {
				response := ErrorResponse{
					Title:  "get_clusters_error",
					Detail: err.Error(),
				}

				elogger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			// No clusters exists
			if clusters == nil {
				response := ErrorResponse{
					Title:  "get_clusters_error",
					Detail: "clusters not found",
				}

				elogger.Error("clusters not found")
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusNotFound)
				return
			}

			logger.Info("returning clusters")
			logger.Debug(clusters)
			respondWithJson(w, newClustersResponse(clusters, context.RequestId()), http.StatusOK)
		})
	}
}

func (ch *ClusterHandler) DeleteCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "delete_cluster",
				"request": context.RequestId(),
			})

			elogger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "handlers",
				"event":   "delete_cluster_error",
				"request": context.RequestId(),
			})

			vars := mux.Vars(r)
			id := vars["id"]

			logger.Info(fmt.Sprintf("delete cluster '%s'", id))

			// Will not continue if missing id in request
			if len(id) <= 0 {
				response := ErrorResponse{
					Title:  "delete_cluster_error",
					Detail: "Missing required cluster id",
				}

				elogger.Error("missing id")
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusBadRequest)
				return
			}

			cluster, err := ch.cs.DeleteCluster(context, id)
			// Internal error in any layer below handler
			if err != nil {
				response := ErrorResponse{
					Title:  "delete_cluster_error",
					Detail: err.Error(),
				}

				elogger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			// Cluster does not exist
			if cluster == nil {
				response := ErrorResponse{
					Title:  "delete_cluster_error",
					Detail: "cluster not found",
				}

				elogger.Error("cluster not found")
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusNotFound)
				return
			}

			logger.Info("deleting cluster")
			logger.Debug(cluster)
			respondWithJson(w, newClusterResponse(cluster, context.RequestId()), http.StatusAccepted)
		})
	}
}

func newClusterResponse(cluster *models.Cluster, request_id string) *RequestResponse {
	cluster_response := ClusterResponse{
		Id:     cluster.Id,
		Name:   cluster.Name,
		Status: cluster.Status,
	}

	response_data := ResponseData{
		Type:       "cluster",
		Attributes: cluster_response,
	}

	request_response := RequestResponse{
		RequestId: request_id,
		Data:      response_data,
	}

	return &request_response
}

func newClustersResponse(clusters []models.Cluster, request_id string) *RequestResponse {

	cluster_list := []ClusterResponse{}

	for _, cluster := range clusters {
		cluster_response := ClusterResponse{
			Id:     cluster.Id,
			Name:   cluster.Name,
			Status: cluster.Status,
		}

		cluster_list = append(cluster_list, cluster_response)
	}

	response_data := ResponseData{
		Type:       "clusters",
		Attributes: cluster_list,
	}

	request_response := RequestResponse{
		RequestId: request_id,
		Data:      response_data,
	}

	return &request_response
}

func respondWithJson(w http.ResponseWriter, response *RequestResponse, status int) {
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// logger.Panic("failed to marshal cluster data for response")
	}

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
