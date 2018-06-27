package daos_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	. "github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/models"
)

var (
	valid_db                 *sqlx.DB
	invalid_db               *sqlx.DB
	config                   app.ServerConfig
	cluster_test_schema      = `CREATE SCHEMA IF NOT EXISTS cluster_test`
	drop_cluster_test_schema = `DROP SCHEMA IF EXISTS cluster_test CASCADE`
	cluster_test_searchpath  = `ALTER ROLE taos SET search_path TO cluster_test, public`
	clusters_ddl             = `
		CREATE TABLE IF NOT EXISTS cluster_test.clusters (
				id 								UUID,
				name 							text,
				status 						text,
				message 					text,
				outputs 					json,
				terraform_state 	json,
				terraform_config 	json,
				timestamp 				timestamp,
				expiration 				timestamp,
				timeout           text,
				project           text,
				region            text
		)`
	truncate_clusters = `TRUNCATE TABLE clusters`
	drop_clusters_ddl = `DROP TABLE IF EXISTS cluster_test.clusters CASCADE`
	create_pgcrypto   = `CREATE EXTENSION pgcrypto`
)

var _ = BeforeSuite(func() {
	// logrus output level while running tests
	log.SetLevel(log.FatalLevel)

	err := app.LoadServerConfig(&config, "../")
	Expect(err).NotTo(HaveOccurred())

	// Useable database connection
	valid_db, err = sqlx.Connect("postgres", config.ConnStr)
	Expect(err).NotTo(HaveOccurred())

	// A closed database connection approximates a
	// non-useable database connection
	invalid_db, err = sqlx.Connect("postgres", config.ConnStr)
	Expect(err).NotTo(HaveOccurred())
	invalid_db.Close()

	// Setup scheme in the useable database connection
	valid_db.MustExec(drop_clusters_ddl)
	valid_db.MustExec(drop_cluster_test_schema)
	valid_db.MustExec(cluster_test_schema)
	valid_db.MustExec(clusters_ddl)
	valid_db.MustExec(cluster_test_searchpath)

})

var _ = AfterSuite(func() {
	valid_db.Close()
})

var _ = Describe("Cluster", func() {

	var (
		cluster                *models.Cluster
		cluster_1              *models.Cluster
		cluster_2              *models.Cluster
		expired_cluster        *models.Cluster
		not_expired_cluster    *models.Cluster
		valid_request_id       string
		valid_timeout          string
		valid_project          string
		valid_region           string
		new_timestamp          time.Time
		new_project            string
		new_region             string
		clusters               []models.Cluster
		err                    error
		dao                    ClusterDao
		tx                     *sqlx.Tx
		valid_terraform_config []byte
	)

	BeforeEach(func() {
		dao = ClusterDao{}

		valid_request_id = "c12c2d58-2af0-11e8-b467-0ed5f89f718b"
		valid_timeout = "10m"
		valid_project = "project_name"
		valid_region = "region_name"

		cluster_1 = &models.Cluster{
			Id:              "a19e2758-0ec5-11e8-ba89-0ed5f89f718b",
			Name:            "cluster_1",
			Status:          "provisioned",
			Message:         "This is a message",
			Outputs:         []byte(`{}`),
			TerraformState:  []byte(`{}`),
			TerraformConfig: []byte(`{}`),
			Timestamp:       time.Now(),
			Timeout:         valid_timeout,
			Project:         valid_project,
			Region:          valid_region,
		}

		cluster_2 = &models.Cluster{
			Id:              "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b",
			Name:            "cluster_2",
			Status:          "provisioned",
			Message:         "This is a message",
			Outputs:         []byte(`{}`),
			TerraformState:  []byte(`{}`),
			TerraformConfig: []byte(`{}`),
			Timestamp:       time.Now(),
			Timeout:         valid_timeout,
			Project:         valid_project,
			Region:          valid_region,
		}

		valid_terraform_config = []byte(`{"provider":{"google":{}}}`)

		// Not expired
		not_expired_cluster = cluster_1
		not_expired_cluster.Timeout = "240h"
		not_expired_cluster.Timestamp = time.Now()
		expiration, time_err := time.ParseDuration(not_expired_cluster.Timeout)
		Expect(time_err).NotTo(HaveOccurred())
		not_expired_cluster.Expiration = not_expired_cluster.Timestamp.Add(time.Duration(expiration))

		// Expired
		expired_cluster = cluster_2
		expired_cluster.Timeout = "10m"
		expired_cluster.Timestamp = time.Now().Add(-15 * time.Minute)
		expiration, time_err = time.ParseDuration(expired_cluster.Timeout)
		Expect(time_err).NotTo(HaveOccurred())
		expired_cluster.Expiration = expired_cluster.Timestamp.Add(time.Duration(expiration))

		// Create a fresh transaction for each test
		tx, err = valid_db.Beginx()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Ensure the test data is removed
		valid_db.MustExec(truncate_clusters)
	})

	// ======================================================================
	//                      _
	//   ___ _ __ ___  __ _| |_ ___
	//  / __| '__/ _ \/ _` | __/ _ \
	// | (__| | |  __/ (_| | ||  __/
	//  \___|_|  \___|\__,_|\__\___|
	//
	// ======================================================================

	Describe("Creating cluster", func() {

		Context("When everything goes ok", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(valid_db, valid_terraform_config, valid_timeout, valid_request_id, valid_project, valid_region)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should have a name", func() {
				Expect(cluster.Name).ToNot(BeEmpty())
			})
			It("Should have the expected status", func() {
				Expect(cluster.Status).To(Equal(models.ClusterStatusRequested))
			})
			It("Should have written the config in the config field", func() {
				Expect(cluster.TerraformConfig).To(Equal(valid_terraform_config))
			})
			It("Should use the request id for the cluster id", func() {
				Expect(cluster.Id).To(Equal(valid_request_id))
			})
			It("Should have a project", func() {
				Expect(cluster.Project).To(Equal(valid_project))
			})
			It("Should have a region", func() {
				Expect(cluster.Region).To(Equal(valid_region))
			})
			It("Should have a timestamp", func() {
				Expect(cluster.Timestamp).NotTo(BeNil())
			})
			It("Should have a timeout", func() {
				Expect(cluster.Timeout).NotTo(BeNil())
			})
			It("Should have an accurate expiration", func() {
				Expect(cluster.Expiration).NotTo(BeNil())
				// The cluster should expire at creation+timeout
				duration, err := time.ParseDuration(cluster.Timeout)
				Expect(err).NotTo(HaveOccurred())
				expiration_time := cluster.Timestamp.Add(duration)
				Expect(cluster.Timestamp).NotTo(BeNil())
				Expect(cluster.Expiration).To(Equal(expiration_time))
			})
		})

		Context("Without terraform configuration", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(valid_db, nil, valid_timeout, valid_request_id, valid_project, valid_region)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("Without a timeout", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(valid_db, valid_terraform_config, "", valid_request_id, valid_project, valid_region)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("Without a request id", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(valid_db, valid_terraform_config, valid_timeout, "", valid_project, valid_region)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("When then database transaction cannot be created", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(invalid_db, nil, valid_timeout, valid_request_id, valid_project, valid_region)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
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

	Describe("Getting a Cluster", func() {

		Context("When everything goes ok", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				cluster, err = dao.GetCluster(valid_db, cluster_1.Id, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should return the expected cluster", func() {
				Expect(cluster.Id).To(Equal(cluster_1.Id))
			})
		})

		Context("When the cluster does not exist", func() {
			BeforeEach(func() {
				// Without inserting any clusters into database
				cluster, err = dao.GetCluster(valid_db, cluster_1.Id, valid_request_id)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("Without a cluster id", func() {
			BeforeEach(func() {
				cluster, err = dao.GetCluster(valid_db, "", valid_request_id)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("Without a request id", func() {
			BeforeEach(func() {
				cluster, err = dao.GetCluster(valid_db, cluster_1.Id, "")
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("When then database transaction cannot be created", func() {
			BeforeEach(func() {
				cluster, err = dao.GetCluster(invalid_db, cluster_1.Id, valid_request_id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
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

	Describe("Retrieving clusters", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				seed_err = seedDatabaseWithCluster(cluster_2)
				Expect(seed_err).NotTo(HaveOccurred())
				clusters, err = dao.GetClusters(valid_db, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return clusters", func() {
				Expect(clusters).To(HaveLen(2))
			})
			It("Should return the expected clusters", func() {
				Expect(clusters).To(HaveLen(2))
				Expect(clusters[0].Id).To(Equal(cluster_1.Id))
				Expect(clusters[1].Id).To(Equal(cluster_2.Id))
			})
		})

		Context("When no clusters exist", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(valid_db, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return an empty list of Clusters", func() {
				Expect(clusters).To(HaveLen(0))
			})
		})

		Context("Without a request id", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(valid_db, "")
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster list", func() {
				Expect(clusters).Should(BeNil())
			})
		})

		Context("When then database transaction cannot be created", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(invalid_db, valid_request_id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster list", func() {
				Expect(clusters).Should(BeNil())
			})
		})

	})

	// ======================================================================
	//                  _       _
	//  _   _ _ __   __| | __ _| |_ ___
	// | | | | '_ \ / _` |/ _` | __/ _ \
	// | |_| | |_) | (_| | (_| | ||  __/
	//  \__,_| .__/ \__,_|\__,_|\__\___|
	//       |_|
	//
	// ======================================================================

	Describe("Updating a clusters status", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "status", "different_status", valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Status).To(Equal("different_status"))
			})
		})

		Context("When updating the status field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "status", "different_status", valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Status).To(Equal("different_status"))
			})
		})

		Context("When updating the message field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "message", "different_message", valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Message).To(Equal("different_message"))
			})
		})

		Context("When updating the outputs field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "outputs", []byte(`{"outputs":{}}`), valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Outputs).To(Equal([]byte(`{"outputs":{}}`)))
			})
		})

		Context("When updating the terraform_config field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "terraform_config", []byte(`{"config":{}}`), valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.TerraformConfig).To(Equal([]byte(`{"config":{}}`)))
			})
		})

		Context("When updating the terraform_state field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "terraform_state", []byte(`{"state":{}}`), valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.TerraformState).To(Equal([]byte(`{"state":{}}`)))
			})
		})

		Context("When updating the timeout field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "timeout", "10h", valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Timeout).To(Equal("10h"))
			})
		})

		Context("When updating the timestamp field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				new_timestamp = time.Now()
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "timestamp", new_timestamp, valid_request_id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Timestamp).NotTo(Equal(new_timestamp))
			})
		})

		Context("When updating the project field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				new_project = "new_project_name"
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "project", new_project, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Project).To(Equal(new_project))
			})
		})

		Context("When updating the region field", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				new_region = "new_region_name"
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "region", new_region, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Region).To(Equal(new_region))
			})
		})

		Context("When updating a field that does not exist", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "not-a-field", "", valid_request_id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When updating a field with the wrong type", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "terraform_config", "not-a-byte-slice", valid_request_id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When nothing is different", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(cluster_1)
				Expect(seed_err).NotTo(HaveOccurred())
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "status", cluster_1.Status, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should not change the field", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := valid_db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster_1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Status).To(Equal(cluster_1.Status))
			})
		})

		Context("When the cluster does not exist", func() {
			BeforeEach(func() {
				err = dao.UpdateClusterField(valid_db, cluster_1.Id, "status", cluster_1.Status, valid_request_id)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
		})

	})

	// ======================================================================
	//             _                     _              _
	//   __ _  ___| |_    _____  ___ __ (_)_ __ ___  __| |
	//  / _` |/ _ \ __|  / _ \ \/ / '_ \| | '__/ _ \/ _` |
	// | (_| |  __/ |_  |  __/>  <| |_) | | | |  __/ (_| |
	//  \__, |\___|\__|  \___/_/\_\ .__/|_|_|  \___|\__,_|
	//  |___/                     |_|
	//
	// ======================================================================

	// Clusters are past their expiration when time.now > (creation.time + timeout)
	Describe("Getting expired clusters", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(expired_cluster)
				Expect(seed_err).NotTo(HaveOccurred())
				seed_err = seedDatabaseWithCluster(not_expired_cluster)
				Expect(seed_err).NotTo(HaveOccurred())
				clusters, err = dao.GetExpiredClusters(valid_db, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return the expected cluster(s)", func() {
				Expect(clusters).To(HaveLen(1))
				Expect(clusters[0].Id).To(Equal(cluster_2.Id))
			})
		})

		Context("When there are no expired clusters", func() {
			BeforeEach(func() {
				seed_err := seedDatabaseWithCluster(not_expired_cluster)
				Expect(seed_err).NotTo(HaveOccurred())
				seed_err = seedDatabaseWithCluster(not_expired_cluster)
				Expect(seed_err).NotTo(HaveOccurred())
				clusters, err = dao.GetExpiredClusters(valid_db, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return no cluster(s)", func() {
				Expect(clusters).To(HaveLen(0))
			})
		})

		Context("When there are expired clusters that are already destroyed or destroying", func() {
			BeforeEach(func() {
				expired_cluster.Status = "destroyed"
				seed_err := seedDatabaseWithCluster(expired_cluster)
				expired_cluster.Status = "destroying"
				seed_err = seedDatabaseWithCluster(expired_cluster)
				Expect(seed_err).NotTo(HaveOccurred())
				clusters, err = dao.GetExpiredClusters(valid_db, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return no cluster(s)", func() {
				Expect(clusters).To(HaveLen(0))
			})
		})

		Context("When there are no clusters", func() {
			BeforeEach(func() {
				clusters, err = dao.GetExpiredClusters(valid_db, valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return no cluster(s)", func() {
				Expect(clusters).To(HaveLen(0))
			})
		})

		Context("When then database transaction cannot be created", func() {
			BeforeEach(func() {
				clusters, err = dao.GetExpiredClusters(invalid_db, valid_request_id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should return no cluster(s)", func() {
				Expect(clusters).To(HaveLen(0))
			})
		})

		Context("Without a request id", func() {
			BeforeEach(func() {
				clusters, err = dao.GetExpiredClusters(valid_db, "")
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should return no cluster(s)", func() {
				Expect(clusters).To(HaveLen(0))
			})
		})

	})

})

func seedDatabaseWithCluster(cluster *models.Cluster) error {
	sql := `INSERT INTO clusters VALUES (
		:id, 
		:name, 
		:status, 
		:message, 
		:outputs, 
		:terraform_config, 
		:terraform_state, 
		:timestamp, 
		:expiration,
		:timeout,
		:project,
		:region
	)`
	_, err := valid_db.NamedExec(sql, cluster)
	return err
}
