package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	. "github.com/kmacoskey/taos"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/handlers"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/terraform"
)

var _ = Describe("Taos", func() {
	var (
		err                        error
		db                         *sqlx.DB
		server                     *http.Server
		response                   *http.Response
		body                       []byte
		cluster_response_json      *handlers.ClusterResponse
		valid_terraform_config     []byte
		expected_terraform_outputs *[]handlers.TerraformOutput
	)

	BeforeSuite(func() {
		valid_terraform_config = []byte(`{"provider":{"google":{"project":"data-gp-toolsmiths","region":"us-central1"}},"output":{"foo":{"value":"bar"}}}`)
		expected_terraform_outputs = &[]handlers.TerraformOutput{
			handlers.TerraformOutput{
				Name:      "foo",
				Sensitive: "true",
				Type:      "foo",
				Value:     "foo",
			},
		}

		err = app.LoadServerConfig(&app.GlobalServerConfig, ".")
		Expect(err).NotTo(HaveOccurred())

		// err = app.InitLogger(app.GlobalServerConfig.Logging)
		// Expect(err).NotTo(HaveOccurred())
		log.SetLevel(log.FatalLevel)

		db, err = app.DatabaseConnect(app.GlobalServerConfig.ConnStr)
		Expect(err).NotTo(HaveOccurred())
		Expect(db).NotTo(BeNil())

		router := mux.NewRouter()
		handlers.ServeClusterResources(router, db)
		server = StartHttpServer(router)

		err = nil
	})

	AfterSuite(func() {
		db.Close()

		err = server.Shutdown(nil)
		Expect(err).NotTo(HaveOccurred())
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
				response, body = httpClusterRequest("PUT", "http://localhost:8080/cluster", valid_terraform_config)
				cluster_response_json = &handlers.ClusterResponse{}
				err = json.Unmarshal(body, &cluster_response_json)
			})
			It("Should not error", func() {
				Expect(response.StatusCode).To(Equal(http.StatusAccepted))
			})
			It("Should return json", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
			})
			It("Should return a request uuid", func() {
				id, err := uuid.Parse(cluster_response_json.RequestId)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeNil())
			})
			It("Should return a cluster", func() {
				Expect(cluster_response_json.Data.Type).To(Equal("cluster"))
			})
			It("Should return a cluster of the same id as the request id", func() {
				Expect(cluster_response_json.Data.Attributes.Id).To(Equal(cluster_response_json.RequestId))
			})
			It("Should return a requested cluster", func() {
				Expect(cluster_response_json.Data.Attributes.Status).To(Equal(models.ClusterStatusRequested))
			})
			It("Should eventually be provisioned", func() {
				Eventually(func() string {
					url := fmt.Sprintf("http://localhost:8080/cluster/%s", cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.Status
				}, 20, .5).Should(Equal(models.ClusterStatusProvisionSuccess))
			})
			It("Should eventually set the message", func() {
				Eventually(func() string {
					url := fmt.Sprintf("http://localhost:8080/cluster/%s", cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.Message
				}, 20, .5).Should(ContainSubstring(terraform.ApplySuccess))
			})
			It("Should eventually set the outputs", func() {
				Eventually(func() []handlers.TerraformOutput {
					url := fmt.Sprintf("http://localhost:8080/cluster/%s", cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.TerraformOutputs
				}, 20, .5).Should(Equal(expected_terraform_outputs))
			})

		})
	})

})

func httpClusterRequest(request_type string, url string, body []byte) (*http.Response, []byte) {
	req, err := http.NewRequest(request_type, url, bytes.NewBuffer(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())
	defer response.Body.Close()

	body, err = ioutil.ReadAll(response.Body)
	Expect(err).NotTo(HaveOccurred())

	return response, body
}
