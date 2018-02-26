package handlers_test

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/kmacoskey/taos/app"
	. "github.com/kmacoskey/taos/handlers"
	"github.com/kmacoskey/taos/models"
)

func emptyhandler(w http.ResponseWriter, r *http.Request) {}

var _ = Describe("Cluster", func() {

	var (
		cluster1                        *models.Cluster
		cluster2                        *models.Cluster
		cluster1_json                   []byte
		cluster_list_json               []byte
		empty_cluster_list_json         []byte
		empty_body_json                 []byte
		cluster1_not_found_error        string
		cluster1_could_not_delete_error string
		response                        *httptest.ResponseRecorder
		err                             error
		resp                            *http.Response
	)

	BeforeEach(func() {
		cluster1 = &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}
		cluster2 = &models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}
		cluster1_json, err = json.Marshal(cluster1)
		cluster1_not_found_error = fmt.Sprintf("could not retrieve cluster '%v'\n", cluster1.Id)
		cluster1_could_not_delete_error = fmt.Sprintf("could not update cluster '%v' status to deleted\n", cluster1.Id)
		Expect(err).NotTo(HaveOccurred())

		response = httptest.NewRecorder()
	})

	// ======================================================================
	//                      _
	//   ___ _ __ ___  __ _| |_ ___
	//  / __| '__/ _ \/ _` | __/ _ \
	// | (__| | |  __/ (_| | ||  __/
	//  \___|_|  \___|\__,_|\__\___|
	//
	// ======================================================================

	Describe("Create a cluster", func() {
		Context("With valid Config", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.GetCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
				request := httptest.NewRequest("POST", "/cluster", bytes.NewBuffer(jsonStr))
				request.Header.Set("Content-Type", "application/json")

				// Create a new request with the expected, but empty, request.Context
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()
			})
			It("Should return a 200 OK", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	// ======================================================================
	//             _
	//   __ _  ___| |_
	//  / _` |/ _ \ __|
	// | (_| |  __/ |_
	//  \__, |\___|\__|
	//  |___/
	//
	// ======================================================================

	Describe("Get a Cluster for a specific id", func() {
		Context("When that cluster exists", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.GetCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/cluster/id", nil)
				request = mux.SetURLVars(request, map[string]string{"id": "1"})
				// Create a new request with the expected, but empty, request.Context
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()
			})
			It("Should return a 200 OK", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
			It("Should return the expected cluster as json in the response body", func() {
				body, err := ioutil.ReadAll(resp.Body)
				cluster := &models.Cluster{}
				err = json.Unmarshal(body, &cluster)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal(cluster1_json))
			})
		})
		Context("When that cluster does not exist", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewEmptyClusterService())
				adapter := ch.GetCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/cluster/id", nil)
				request = mux.SetURLVars(request, map[string]string{"id": cluster1.Id})
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()
			})
			It("Should return a 404 Not Found", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
			It("Should return nothing in the response body", func() {
				body, _ := ioutil.ReadAll(resp.Body)
				Expect(string(body)).To(Equal(cluster1_not_found_error))
			})
		})
	})

	// ======================================================================
	//             _
	//   __ _  ___| |_ ___
	//  / _` |/ _ \ __/ __|
	// | (_| |  __/ |_\__ \
	//  \__, |\___|\__|___/
	//  |___/
	//
	// ======================================================================

	Describe("Get all clusters", func() {
		Context("When Clusters Exist", func() {
			BeforeEach(func() {
				// Create a ClusterList of valid Clusters
				clusters := []*models.Cluster{}
				clusters = append(clusters, cluster1)
				clusters = append(clusters, cluster2)
				cluster_list := &ClusterList{TotalCount: 2, Clusters: clusters}
				cluster_list_json, err = json.Marshal(cluster_list)
				Expect(err).NotTo(HaveOccurred())

				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.GetClusters()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/clusters", nil)
				// Create a new request with the expected, but empty, request.Context
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()
			})
			It("Should return a 200 OK", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
			It("Should return all cluseters", func() {
				body, err := ioutil.ReadAll(resp.Body)
				cluster := &ClusterList{}
				err = json.Unmarshal(body, &cluster)
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(Equal(cluster_list_json))
			})
		})
		Context("When No Clusters Exist", func() {
			BeforeEach(func() {
				// Create a ClusterList of no Clusters
				empty_clusters := []*models.Cluster{}
				empty_cluster_list := &ClusterList{TotalCount: 0, Clusters: empty_clusters}
				empty_cluster_list_json, err = json.Marshal(empty_cluster_list)
				Expect(err).NotTo(HaveOccurred())

				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewEmptyClusterService())
				adapter := ch.GetClusters()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/clusters", nil)
				// Create a new request with the expected, but empty, request.Context
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()
			})
			It("Should return a 200 OK", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
			It("Should return an empty slice of Clusters in a ClusterList", func() {
				body, err := ioutil.ReadAll(resp.Body)
				cluster := &ClusterList{}
				err = json.Unmarshal(body, &cluster)
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(Equal(empty_cluster_list_json))
			})
		})
	})

	Describe("Delete a Cluster for a specific id", func() {
		Context("When that cluster exists", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.DeleteCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				// Create a new request with the expected, but empty, request.Context
				request := httptest.NewRequest("DELETE", "/cluster/id", nil)
				request = mux.SetURLVars(request, map[string]string{"id": "1"})
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()
			})
			It("Should return a 200 OK", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
			It("Should return nothing in the response body", func() {
				body, _ := ioutil.ReadAll(resp.Body)
				Expect(body).To(Equal(empty_body_json))
			})
		})
		Context("When that cluster does not exist", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewEmptyClusterService())
				adapter := ch.DeleteCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				// Create a new request with the expected, but empty, request.Context
				request := httptest.NewRequest("DELETE", "/cluster/id", nil)
				request = mux.SetURLVars(request, map[string]string{"id": cluster1.Id})
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()
			})
			It("Should return a 404 Not Found", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
			It("Should return nothing in the response body", func() {
				body, _ := ioutil.ReadAll(resp.Body)
				Expect(string(body)).To(Equal(cluster1_could_not_delete_error))
			})
		})
	})

})

/*
 * Valid Cluster Service returns valid Clusters
 */
type ValidClusterService struct{}

func NewValidClusterService() *ValidClusterService {
	return &ValidClusterService{}
}

func (cs *ValidClusterService) CreateCluster(rc app.RequestContext) (*models.Cluster, error) {
	return &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}, nil
}

func (cs *ValidClusterService) GetCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	return &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}, nil
}

func (cs *ValidClusterService) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	cluster1 := models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}
	cluster2 := models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}
	clusters = append(clusters, cluster1)
	clusters = append(clusters, cluster2)
	return clusters, nil
}

func (cs *ValidClusterService) DeleteCluster(rc app.RequestContext, id string) error {
	return nil
}

/*
 * Empty Cluster Service returns no Clusters
 */
type EmptyClusterService struct{}

func NewEmptyClusterService() *EmptyClusterService {
	return &EmptyClusterService{}
}

func (cs *EmptyClusterService) CreateCluster(rc app.RequestContext) (*models.Cluster, error) {
	return nil, nil
}

func (cs *EmptyClusterService) GetCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	return nil, errors.New(fmt.Sprintf("could not retrieve cluster '%v'", id))
}

func (cs *EmptyClusterService) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	return []models.Cluster{}, nil
}

func (cs *EmptyClusterService) DeleteCluster(rc app.RequestContext, id string) error {
	return errors.New(fmt.Sprintf("could not update cluster '%s' status to deleted", id))
}
