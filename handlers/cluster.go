package handlers

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/services"
	"github.com/kmacoskey/taos/terraform"
	log "github.com/sirupsen/logrus"
)

type clusterService interface {
	GetCluster(request_id string, id string) (*models.Cluster, error)
	GetClusters(request_id string) ([]models.Cluster, error)
	CreateCluster(terraform_config []byte, timeout string, request_id string, client services.TerraformClient) (*models.Cluster, error)
	DeleteCluster(request_id string, client services.TerraformClient, id string) (*models.Cluster, error)
}

type ClusterHandler struct {
	service clusterService
}

func NewClusterHandler(service clusterService) *ClusterHandler {
	return &ClusterHandler{service}
}

func ServeClusterResources(router *mux.Router, db *sqlx.DB) {
	handler := NewClusterHandler(services.NewClusterService(daos.NewClusterDao(), db))

	router.Handle("/cluster/{id}", app.Adapt(
		router,
		handler.GetCluster(),
		app.WithRequestContext(),
	)).Methods("GET")

	router.Handle("/clusters", app.Adapt(
		router,
		handler.GetClusters(),
		app.WithRequestContext(),
	)).Methods("GET")

	router.Handle("/cluster", app.Adapt(
		router,
		handler.CreateCluster(),
		app.WithRequestContext(),
	)).Methods("PUT")

	router.Handle("/cluster/{id}", app.Adapt(
		router,
		handler.DeleteCluster(),
		app.WithRequestContext(),
	)).Methods("DELETE")
}

func getBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ch *ClusterHandler) CreateCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{"package": "handlers", "event": "create_cluster", "request": context.RequestId()})

			body, err := ioutil.ReadAll(r.Body)

			if err != nil {
				response := ErrorResponseAttributes{Title: "create_cluster_error", Detail: err.Error()}
				logger.Error(err)
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusBadRequest)
				return
			}

			if len(body) <= 0 {
				err := errors.New("Missing required terraform configuration for create cluster request")
				response := ErrorResponseAttributes{Title: "create_cluster_error", Detail: err.Error()}
				logger.Error("missing terraform config")
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusBadRequest)
				return
			}

			cluster_request := ClusterRequest{}
			err = json.Unmarshal(body, &cluster_request)
			if err != nil {
				logger.Error(err)
				return
			}

			cluster, err := ch.service.CreateCluster([]byte(cluster_request.TerraformConfig), cluster_request.Timeout, context.RequestId(), terraform.NewTerraformClient())

			// Currently no expectation for the situation that
			// err == nil && cluster == nil
			// If a cluster is not returned, then an err has occured
			// Eventually this may capture the situation where resources are not available
			if err != nil || cluster == nil {
				response := ErrorResponseAttributes{Title: "create_cluster_error", Detail: err.Error()}
				logger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			respondWithJson(w, newClusterResponse(cluster, context.RequestId()), http.StatusAccepted)
		})
	}
}

// Retrieve a single Cluster for a given id
func (ch *ClusterHandler) GetCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{"package": "handlers", "event": "get_cluster", "request": context.RequestId()})

			vars := mux.Vars(r)
			id := vars["id"]

			if len(id) <= 0 {
				err := errors.New("missing required cluster id")
				response := ErrorResponseAttributes{Title: "get_cluster_error", Detail: err.Error()}
				logger.Error(err)
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusBadRequest)
				return
			}

			cluster, err := ch.service.GetCluster(context.RequestId(), id)
			if err != nil {
				response := ErrorResponseAttributes{Title: "get_cluster_error", Detail: err.Error()}
				logger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			// Cluster does not exist
			if cluster == nil {
				err := errors.New("cluster not found")
				response := ErrorResponseAttributes{Title: "get_cluster_error", Detail: err.Error()}
				logger.Error(err)
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusNotFound)
				return
			}

			respondWithJson(w, newClusterResponse(cluster, context.RequestId()), http.StatusOK)
		})
	}
}

// Retrieve a ClusterList of all Clusters
func (ch *ClusterHandler) GetClusters() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{"package": "handlers", "event": "get_clusters", "request": context.RequestId()})

			clusters, err := ch.service.GetClusters(context.RequestId())
			if err != nil {
				response := ErrorResponseAttributes{Title: "get_clusters_error", Detail: err.Error()}
				logger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			respondWithJson(w, newClustersResponse(clusters, context.RequestId()), http.StatusOK)
		})
	}
}

func (ch *ClusterHandler) DeleteCluster() app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			context := app.GetRequestContext(r)

			logger := log.WithFields(log.Fields{"package": "handlers", "event": "delete_cluster", "request": context.RequestId()})

			vars := mux.Vars(r)
			id := vars["id"]

			if len(id) <= 0 {
				err := errors.New("missing required cluster id")
				response := ErrorResponseAttributes{Title: "delete_cluster_error", Detail: err.Error()}
				logger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusBadRequest)
				return
			}

			cluster, err := ch.service.DeleteCluster(context.RequestId(), terraform.NewTerraformClient(), id)
			if err != nil {
				response := ErrorResponseAttributes{Title: "delete_cluster_error", Detail: err.Error()}
				logger.Error(err.Error())
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusInternalServerError)
				return
			}

			if cluster == nil {
				err := errors.New("cluster not found")
				response := ErrorResponseAttributes{Title: "delete_cluster_error", Detail: err.Error()}
				logger.Error(err)
				respondWithJson(w, newErrorResponse(&response, context.RequestId()), http.StatusNotFound)
				return
			}

			respondWithJson(w, newClusterResponse(cluster, context.RequestId()), http.StatusAccepted)
		})
	}
}

func newClusterResponse(cluster *models.Cluster, request_id string) *ClusterResponse {
	logger := log.WithFields(log.Fields{"package": "handlers", "event": "cluster_response", "request": request_id})

	var outputs map[string]TerraformOutput
	if cluster.Outputs != nil {
		err := json.Unmarshal(cluster.Outputs, &outputs)
		if err != nil {
			logger.Error(err)
			return nil
		}
	}

	cluster_response := ClusterResponseAttributes{
		Id:               cluster.Id,
		Name:             cluster.Name,
		Status:           cluster.Status,
		Message:          cluster.Message,
		TerraformOutputs: outputs,
	}

	response_data := ClusterResponseData{Type: "cluster", Attributes: cluster_response}
	request_response := ClusterResponse{RequestId: request_id, Data: response_data}

	return &request_response
}

func newClustersResponse(clusters []models.Cluster, request_id string) *ClustersResponse {
	logger := log.WithFields(log.Fields{"package": "handlers", "event": "clusters_response", "request": request_id})

	cluster_list := []ClusterResponseAttributes{}

	for _, cluster := range clusters {
		var outputs map[string]TerraformOutput
		if cluster.Outputs != nil {
			err := json.Unmarshal(cluster.Outputs, &outputs)
			if err != nil {
				logger.Error(err)
				return nil
			}
		}

		cluster_response := ClusterResponseAttributes{
			Id:               cluster.Id,
			Name:             cluster.Name,
			Status:           cluster.Status,
			Message:          cluster.Message,
			TerraformOutputs: outputs,
		}

		cluster_list = append(cluster_list, cluster_response)
	}

	response_data := ClustersResponseData{Type: "clusters", Attributes: cluster_list}
	request_response := ClustersResponse{RequestId: request_id, Data: response_data}

	return &request_response
}

func newErrorResponse(response *ErrorResponseAttributes, request_id string) *ErrorResponse {
	response_data := ErrorResponseData{Type: "error", Attributes: response}
	request_response := ErrorResponse{RequestId: request_id, Data: response_data}
	return &request_response
}

func respondWithJson(w http.ResponseWriter, response interface{}, status int) {
	logger := log.WithFields(log.Fields{"package": "handlers", "event": "json_response"})

	js, err := json.Marshal(response)
	if err != nil {
		logger.Error(err)
	}

	// Set() header then WriteHeader(), the order does matter
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
}
