package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	. "github.com/kmacoskey/taos"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/handlers"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/reaper"
	"github.com/kmacoskey/taos/services"
	"github.com/kmacoskey/taos/terraform"
)

var _ = Describe("Taos", func() {
	var (
		err                        error
		db                         *sqlx.DB
		server                     *http.Server
		server_port                string
		response                   *http.Response
		body                       []byte
		cluster_id                 string
		cluster_response_json      *handlers.ClusterResponse
		valid_terraform_config     []byte
		expected_terraform_outputs map[string]handlers.TerraformOutput
	)

	BeforeSuite(func() {
		// err = app.InitLogger(app.GlobalServerConfig.Logging)
		// Expect(err).NotTo(HaveOccurred())
		log.SetLevel(log.FatalLevel)

		valid_terraform_config = []byte(`{"config":"{\"provider\":{\"google\":{\"project\":\"data-gp-toolsmiths\",\"region\":\"us-central1\"}},\"output\":{\"foo\":{\"value\":\"bar\"}}}","timeout":"5s"}`)

		expected_terraform_outputs = make(map[string]handlers.TerraformOutput)
		expected_terraform_outputs["foo"] = handlers.TerraformOutput{
			Sensitive: false,
			Type:      "string",
			Value:     "bar",
		}
		err = app.LoadServerConfig(&app.GlobalServerConfig, ".")
		Expect(err).NotTo(HaveOccurred())

		server_port = app.GlobalServerConfig.ServerPort
		// Run the server ListenAndServer in a go thread to allow for testing
		app.GlobalServerConfig.BackgroundForTesting = true

		db, err = app.DatabaseConnect(app.GlobalServerConfig.ConnStr)
		Expect(err).NotTo(HaveOccurred())
		Expect(db).NotTo(BeNil())

		// Ensure a clean table of clusters before testing
		truncate_clusters := `TRUNCATE TABLE clusters`
		db.MustExec(truncate_clusters)

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
				response, body = httpClusterRequest("PUT", fmt.Sprintf("http://localhost:%s/cluster", server_port), valid_terraform_config)
				cluster_response_json = &handlers.ClusterResponse{}
				err = json.Unmarshal(body, &cluster_response_json)
			})
			// Note, this test is a bit loaded in it's concerns
			//  in order to limit the individual specs. It is slow
			//  when testing actually runs terraform.
			It("Should return the expected cluster", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusAccepted))
				Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
				Expect(cluster_response_json.Data.Type).To(Equal("cluster"))
				Expect(cluster_response_json.Data.Attributes.Status).To(Equal(models.ClusterStatusRequested))
			})
			It("Should eventually be provisioned", func() {
				Eventually(func() string {
					url := fmt.Sprintf("http://localhost:%s/cluster/%s", server_port, cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.Status
				}, 40, .5).Should(Equal(models.ClusterStatusProvisionSuccess))
			})
			It("Should eventually set the message", func() {
				Eventually(func() string {
					url := fmt.Sprintf("http://localhost:%s/cluster/%s", server_port, cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.Message
				}, 40, .5).Should(ContainSubstring(terraform.ApplySuccess))
			})
			It("Should eventually set the outputs", func() {
				Eventually(func() map[string]handlers.TerraformOutput {
					url := fmt.Sprintf("http://localhost:%s/cluster/%s", server_port, cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.TerraformOutputs
				}, 40, .5).Should(Equal(expected_terraform_outputs))
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
	Describe("Deleting a cluster", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				response, body = httpClusterRequest("PUT", fmt.Sprintf("http://localhost:%s/cluster", server_port), valid_terraform_config)
				temp_cluster_response_json := &handlers.ClusterResponse{}
				err = json.Unmarshal(body, &temp_cluster_response_json)
				Expect(err).NotTo(HaveOccurred())

				cluster_id = temp_cluster_response_json.Data.Attributes.Id
				time.Sleep(40 * time.Second)

				response, body = httpClusterRequest("DELETE", fmt.Sprintf("http://localhost:%s/cluster/%s", server_port, cluster_id), nil)
				cluster_response_json = &handlers.ClusterResponse{}
				err = json.Unmarshal(body, &cluster_response_json)
			})
			// Note, this test is a bit loaded in it's concerns
			//  in order to limit the individual specs. It is slow
			//  when testing actually runs terraform.
			It("Should return a successfully cluster", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusAccepted))
				Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
				Expect(cluster_response_json.Data.Type).To(Equal("cluster"))
			})
			It("Should eventually be destroyed", func() {
				Eventually(func() string {
					url := fmt.Sprintf("http://localhost:%s/cluster/%s", server_port, cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.Status
				}, 40, .5).Should(Equal(models.ClusterStatusDestroyed))
			})
		})
	})

	// ======================================================================
	//  _ __ ___  __ _ _ __
	// | '__/ _ \/ _` | '_ \
	// | | |  __/ (_| | |_) |
	// |_|  \___|\__,_| .__/
	//                |_|
	// ======================================================================
	Describe("Automatic reaping", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				reaper, _ := reaper.NewClusterReaper("5s", services.NewClusterService(daos.NewClusterDao(), db), db)
				reaper.StartReaping()

				response, body = httpClusterRequest("PUT", fmt.Sprintf("http://localhost:%s/cluster", server_port), valid_terraform_config)
				cluster_response_json = &handlers.ClusterResponse{}
				err = json.Unmarshal(body, &cluster_response_json)
				Expect(err).NotTo(HaveOccurred())

				// Give time to provision the cluster
				time.Sleep(10 * time.Second)
			})
			It("Should eventually reap expired clusters", func() {
				Eventually(func() string {
					url := fmt.Sprintf("http://localhost:%s/cluster/%s", server_port, cluster_response_json.Data.Attributes.Id)

					_, eventual_body := httpClusterRequest("GET", url, valid_terraform_config)
					eventual_cluster_response_json := &handlers.ClusterResponse{}
					err = json.Unmarshal(eventual_body, &eventual_cluster_response_json)
					Expect(err).NotTo(HaveOccurred())

					return eventual_cluster_response_json.Data.Attributes.Status
				}, 100, 5).Should(Equal(models.ClusterStatusDestroyed))

			})
		})
	})

})

func httpClusterRequest(request_type string, url string, body []byte) (*http.Response, []byte) {
	req, err := http.NewRequest(request_type, url, bytes.NewBuffer(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	client := &http.Client{}
	response, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())
	defer response.Body.Close()

	body, err = ioutil.ReadAll(response.Body)
	Expect(err).NotTo(HaveOccurred())

	return response, body
}
