package handlers_test

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kmacoskey/taos/app"
	. "github.com/kmacoskey/taos/handlers"
	"github.com/kmacoskey/taos/models"
)

func emptyhandler(w http.ResponseWriter, r *http.Request) {}

var _ = Describe("Cluster", func() {

	var (
		cluster1                        *models.Cluster
		cluster1Response                ClusterResponseAttributes
		cluster2                        *models.Cluster
		cluster1_json                   []byte
		cluster1_not_found_error        string
		cluster1_could_not_delete_error string
		response                        *httptest.ResponseRecorder
		err                             error
		json_err                        error
		resp                            *http.Response
		body                            []byte
		cluster_response_json           *ClusterResponse
		clusters_response_json          *ClustersResponse
		error_response_json             *ErrorResponse
	)

	BeforeEach(func() {
		log.SetLevel(log.FatalLevel)
		var outputsBlob = []byte(`[{"name":"foobar","sensitive":"true","type":"foo","value":"bar"},{"name":"barfoo","sensitive":"false","type":"bar","value":"foo"}]`)
		var outputs = []TerraformOutput{
			TerraformOutput{
				Name:      "foobar",
				Sensitive: "true",
				Type:      "foo",
				Value:     "bar",
			},
			TerraformOutput{
				Name:      "barfoo",
				Sensitive: "false",
				Type:      "bar",
				Value:     "foo",
			},
		}

		cluster1 = &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status", Outputs: outputsBlob}
		cluster1Response = ClusterResponseAttributes{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status", Message: "", TerraformOutputs: outputs}
		cluster2 = &models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status", Outputs: outputsBlob}

		cluster1_json, err = json.Marshal(cluster1)
		Expect(err).NotTo(HaveOccurred())
		cluster1_not_found_error = fmt.Sprintf("could not retrieve cluster '%v'\n", cluster1.Id)
		cluster1_could_not_delete_error = fmt.Sprintf("could not update cluster '%v' status to deleted\n", cluster1.Id)
	})

	// ======================================================================
	//                      _
	//   ___ _ __ ___  __ _| |_ ___
	//  / __| '__/ _ \/ _` | __/ _ \
	// | (__| | |  __/ (_| | ||  __/
	//  \___|_|  \___|\__,_|\__\___|
	//
	// ======================================================================

	Describe("Creating a cluster", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.CreateCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
				request := httptest.NewRequest("POST", "/cluster", bytes.NewBuffer(jsonStr))
				request.Header.Set("Content-Type", "application/json")

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				cluster_response_json = &ClusterResponse{}
				json_err = json.Unmarshal(body, &cluster_response_json)
			})
			It("Should return a 202 OK", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(cluster_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return a cluster", func() {
				Expect(cluster_response_json.Data.Type).To(Equal("cluster"))
			})
			It("should return the expected terraform outputs", func() {
				Expect(cluster_response_json.Data.Attributes).To(Equal(cluster1Response))
			})
		})

		Context("When the Cluster service, daos, or terraform service has errored", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewErroringClusterService())
				adapter := ch.CreateCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
				request := httptest.NewRequest("POST", "/cluster", bytes.NewBuffer(jsonStr))
				request.Header.Set("Content-Type", "application/json")

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				error_response_json = &ErrorResponse{}
				json_err = json.Unmarshal(body, &error_response_json)
			})
			It("Should return a 500", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(error_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return an error", func() {
				Expect(error_response_json.Data.Type).To(Equal("error"))
			})
		})

		Context("When no terraform config is included", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.CreateCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				var jsonStr = []byte(``)
				request := httptest.NewRequest("POST", "/cluster", bytes.NewBuffer(jsonStr))
				request.Header.Set("Content-Type", "application/json")

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				error_response_json = &ErrorResponse{}
				json_err = json.Unmarshal(body, &error_response_json)
			})
			It("Should return a 400", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(error_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return an error", func() {
				Expect(error_response_json.Data.Type).To(Equal("error"))
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
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.GetCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/cluster/id", nil)
				request = mux.SetURLVars(request, map[string]string{"id": "1"})

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				cluster_response_json = &ClusterResponse{}
				json_err = json.Unmarshal(body, &cluster_response_json)
			})
			It("Should return a 200", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(cluster_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return a cluster", func() {
				Expect(cluster_response_json.Data.Type).To(Equal("cluster"))
			})
			It("Should return the expected cluster", func() {
				cr := cluster_response_json.Data.Attributes
				Expect(cr.Id).To(Equal(cluster1.Id))
			})
		})

		Context("When the handler, service, or daos errors", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewErroringClusterService())
				adapter := ch.GetCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/cluster/id", nil)
				request = mux.SetURLVars(request, map[string]string{"id": "1"})

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				error_response_json = &ErrorResponse{}
				json_err = json.Unmarshal(body, &error_response_json)
			})
			It("Should return a 500", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(error_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return a error", func() {
				Expect(error_response_json.Data.Type).To(Equal("error"))
			})
		})

		Context("When the requested cluster id does not exist", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewEmptyClusterService())
				adapter := ch.GetCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/cluster/id", nil)
				request = mux.SetURLVars(request, map[string]string{"id": "1"})

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				error_response_json = &ErrorResponse{}
				json_err = json.Unmarshal(body, &error_response_json)
			})
			It("Should return a 404", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(error_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return a error", func() {
				Expect(error_response_json.Data.Type).To(Equal("error"))
			})
		})

		Context("When an id was not included in the request", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewEmptyClusterService())
				adapter := ch.GetCluster()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/cluster", nil)

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				error_response_json = &ErrorResponse{}
				json_err = json.Unmarshal(body, &error_response_json)
			})
			It("Should return a 400", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(error_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return a error", func() {
				Expect(error_response_json.Data.Type).To(Equal("error"))
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
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewValidClusterService())
				adapter := ch.GetClusters()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/clusters", nil)

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				clusters_response_json = &ClustersResponse{}
				json_err = json.Unmarshal(body, &clusters_response_json)
			})
			It("Should return a 200", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(clusters_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return clusters", func() {
				Expect(clusters_response_json.Data.Type).To(Equal("clusters"))
			})
			It("Should return the expected clusters", func() {
				cr := clusters_response_json.Data.Attributes
				cr1 := cr[0]
				cr2 := cr[1]
				Expect(cr1.Id).To(Equal(cluster1.Id))
				Expect(cr2.Id).To(Equal(cluster2.Id))
			})
		})

		Context("When there are no clusters to return", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewEmptyClusterService())
				adapter := ch.GetClusters()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/clusters", nil)

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				clusters_response_json = &ClustersResponse{}
				json_err = json.Unmarshal(body, &clusters_response_json)
			})
			It("Should return a 200", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(clusters_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return clusters", func() {
				Expect(clusters_response_json.Data.Type).To(Equal("clusters"))
			})
			It("Should return no clusters", func() {
				cr := clusters_response_json.Data.Attributes
				Expect(cr).To(BeEmpty())
			})
		})

		Context("When the handler, service, or daos errors", func() {
			BeforeEach(func() {
				// Unravel the middleware pattern to test only the Handler
				ch := NewClusterHandler(NewErroringClusterService())
				adapter := ch.GetClusters()
				handler := adapter(http.HandlerFunc(emptyhandler))

				request := httptest.NewRequest("GET", "/clusters", nil)

				// Create a new request with the expected, but empty, request.Context
				response = httptest.NewRecorder()
				requestContext := app.NewRequestContext(request.Context(), request)
				ctx := context.WithValue(request.Context(), "request", requestContext)

				// Create a server to get receive a response for the given request
				handler.ServeHTTP(response, request.WithContext(ctx))
				resp = response.Result()

				// Read the response body
				body, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				error_response_json = &ErrorResponse{}
				json_err = json.Unmarshal(body, &error_response_json)
			})
			It("Should return a 500", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})
			It("Should return json", func() {
				Expect(json_err).NotTo(HaveOccurred())
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(error_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return an error", func() {
				Expect(error_response_json.Data.Type).To(Equal("error"))
			})
		})

	})

	// ======================================================================
	//      _      _      _
	//   __| | ___| | ___| |_ ___
	//  / _` |/ _ \ |/ _ \ __/ _ \
	// | (_| |  __/ |  __/ ||  __/
	//  \__,_|\___|_|\___|\__\___|
	//
	// ======================================================================

	Context("When everything goes ok", func() {
		BeforeEach(func() {
			// Unravel the middleware pattern to test only the Handler
			ch := NewClusterHandler(NewValidClusterService())
			adapter := ch.DeleteCluster()
			handler := adapter(http.HandlerFunc(emptyhandler))

			// Create a new request with the expected, but empty, request.Context
			request := httptest.NewRequest("DELETE", "/cluster/id", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "1"})

			// Create a new request with the expected, but empty, request.Context
			response = httptest.NewRecorder()
			requestContext := app.NewRequestContext(request.Context(), request)
			ctx := context.WithValue(request.Context(), "request", requestContext)

			// Create a server to get receive a response for the given request
			handler.ServeHTTP(response, request.WithContext(ctx))
			resp = response.Result()

			// Read the response body
			body, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			cluster_response_json = &ClusterResponse{}
			json_err = json.Unmarshal(body, &cluster_response_json)
		})
		It("Should return a 202", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
		})
		It("Should return json", func() {
			Expect(json_err).NotTo(HaveOccurred())
		})
		It("Should return a request uuid", func() {
			id, err := uuid.Parse(cluster_response_json.RequestId)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeNil())
		})
		It("Should return a cluster", func() {
			Expect(cluster_response_json.Data.Type).To(Equal("cluster"))
		})
		It("Should return the expected cluster", func() {
			cr := cluster_response_json.Data.Attributes
			Expect(cr.Id).To(Equal(cluster1.Id))
		})
	})

	Context("When the handler, service, or daos errors", func() {
		BeforeEach(func() {
			// Unravel the middleware pattern to test only the Handler
			ch := NewClusterHandler(NewErroringClusterService())
			adapter := ch.DeleteCluster()
			handler := adapter(http.HandlerFunc(emptyhandler))

			// Create a new request with the expected, but empty, request.Context
			request := httptest.NewRequest("DELETE", "/cluster/id", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "1"})

			// Create a new request with the expected, but empty, request.Context
			response = httptest.NewRecorder()
			requestContext := app.NewRequestContext(request.Context(), request)
			ctx := context.WithValue(request.Context(), "request", requestContext)

			// Create a server to get receive a response for the given request
			handler.ServeHTTP(response, request.WithContext(ctx))
			resp = response.Result()

			// Read the response body
			body, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			error_response_json = &ErrorResponse{}
			json_err = json.Unmarshal(body, &error_response_json)
		})
		It("Should return a 500", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		})
		It("Should return json", func() {
			Expect(json_err).NotTo(HaveOccurred())
		})
		It("Should return a request uuid", func() {
			id, err := uuid.Parse(error_response_json.RequestId)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeNil())
		})
		It("Should return a error", func() {
			Expect(error_response_json.Data.Type).To(Equal("error"))
		})
	})

	Context("When the cluster does not exist", func() {
		BeforeEach(func() {
			// Unravel the middleware pattern to test only the Handler
			ch := NewClusterHandler(NewEmptyClusterService())
			adapter := ch.DeleteCluster()
			handler := adapter(http.HandlerFunc(emptyhandler))

			// Create a new request with the expected, but empty, request.Context
			request := httptest.NewRequest("DELETE", "/cluster/id", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "1"})

			// Create a new request with the expected, but empty, request.Context
			response = httptest.NewRecorder()
			requestContext := app.NewRequestContext(request.Context(), request)
			ctx := context.WithValue(request.Context(), "request", requestContext)

			// Create a server to get receive a response for the given request
			handler.ServeHTTP(response, request.WithContext(ctx))
			resp = response.Result()

			// Read the response body
			body, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			error_response_json = &ErrorResponse{}
			json_err = json.Unmarshal(body, &error_response_json)
		})
		It("Should return a 404", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
		It("Should return json", func() {
			Expect(json_err).NotTo(HaveOccurred())
		})
		It("Should return a request uuid", func() {
			id, err := uuid.Parse(error_response_json.RequestId)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeNil())
		})
		It("Should return a error", func() {
			Expect(error_response_json.Data.Type).To(Equal("error"))
		})
	})

	Context("When the id is not included in the request", func() {
		BeforeEach(func() {
			// Unravel the middleware pattern to test only the Handler
			ch := NewClusterHandler(NewEmptyClusterService())
			adapter := ch.DeleteCluster()
			handler := adapter(http.HandlerFunc(emptyhandler))

			// Create a new request with the expected, but empty, request.Context
			request := httptest.NewRequest("DELETE", "/cluster", nil)

			// Create a new request with the expected, but empty, request.Context
			response = httptest.NewRecorder()
			requestContext := app.NewRequestContext(request.Context(), request)
			ctx := context.WithValue(request.Context(), "request", requestContext)

			// Create a server to get receive a response for the given request
			handler.ServeHTTP(response, request.WithContext(ctx))
			resp = response.Result()

			// Read the response body
			body, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			error_response_json = &ErrorResponse{}
			json_err = json.Unmarshal(body, &error_response_json)
		})
		It("Should return a 400", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
		It("Should return json", func() {
			Expect(json_err).NotTo(HaveOccurred())
		})
		It("Should return a request uuid", func() {
			id, err := uuid.Parse(error_response_json.RequestId)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeNil())
		})
		It("Should return a error", func() {
			Expect(error_response_json.Data.Type).To(Equal("error"))
		})
	})

	/*
	 * When the cluster is already deleted
	 *   It should return a 409
	 *   It should return json
	 *   The json returned should be
	 *     {
	 *       "request_id": "550e8400-e29b-41d4-a716-446655440000",
	 *       "status": "409 Conflict",
	 *       "data": {
	 *         "type": "error",
	 *         "attributes": {
	 *           "title": "Error while deleting cluster",
	 *           "detail": "clusted is already deleted"
	 *         }
	 *       }
	 *     }
	 */

	/*
	 * When the cluster is already deleting
	 *   It should return a 409
	 *   It should return json
	 *   The json returned should be
	 *     {
	 *       "request_id": "550e8400-e29b-41d4-a716-446655440000",
	 *       "status": "409 Conflict",
	 *       "data": {
	 *         "type": "error",
	 *         "attributes": {
	 *           "title": "Error while deleting cluster",
	 *           "detail": "clusted is already deleting"
	 *         }
	 *       }
	 *     }
	 */

})

/*
 * Valid Cluster Service returns valid Clusters
 */
type ValidClusterService struct{}

func NewValidClusterService() *ValidClusterService {
	return &ValidClusterService{}
}

func (cs *ValidClusterService) CreateCluster(rc app.RequestContext) (*models.Cluster, error) {
	var outputsBlob = []byte(`[{"name":"foobar","sensitive":"true","type":"foo","value":"bar"},{"name":"barfoo","sensitive":"false","type":"bar","value":"foo"}]`)

	cluster1 := &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status", Outputs: outputsBlob}
	return cluster1, nil
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

func (cs *ValidClusterService) DeleteCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	cluster1 := models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}
	return &cluster1, nil
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
	return nil, nil
}

func (cs *EmptyClusterService) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	return []models.Cluster{}, nil
}

func (cs *EmptyClusterService) DeleteCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	return nil, nil
}

/*
 * Erroring Cluster Service returns that the Cluster Service has errored
 */
type ErroringClusterService struct{}

func NewErroringClusterService() *ErroringClusterService {
	return &ErroringClusterService{}
}

func (cs *ErroringClusterService) CreateCluster(rc app.RequestContext) (*models.Cluster, error) {
	return nil, errors.New("Cluster service error")
}

func (cs *ErroringClusterService) GetCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	return nil, errors.New("Cluster service error")
}

func (cs *ErroringClusterService) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	return nil, errors.New("Cluster service error")
}

func (cs *ErroringClusterService) DeleteCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	return nil, errors.New("Cluster service error")
}
