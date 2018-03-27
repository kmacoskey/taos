package daos_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	. "github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/models"
)

var (
	db                       *sqlx.DB
	InvalidDB                *sqlx.DB
	config                   app.ServerConfig
	cluster_test_schema      = `CREATE SCHEMA IF NOT EXISTS cluster_test`
	drop_cluster_test_schema = `DROP SCHEMA IF EXISTS cluster_test CASCADE`
	cluster_test_searchpath  = `ALTER ROLE taos SET search_path TO cluster_test, public`
	clusters_ddl             = `
		CREATE TABLE IF NOT EXISTS cluster_test.clusters (
				id 								UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
				name 							text,
				status 						text,
				message 					text,
				outputs 					text,
				terraform_state 	text,
				terraform_config 	json
		)`
	truncate_clusters = `TRUNCATE TABLE clusters`
	drop_clusters_ddl = `DROP TABLE IF EXISTS cluster_test.clusters CASCADE`
	create_pgcrypto   = `CREATE EXTENSION pgcrypto`
)

var _ = BeforeSuite(func() {
	log.SetLevel(log.FatalLevel)

	err := app.LoadServerConfig(&config, "../")
	Expect(err).NotTo(HaveOccurred())

	// A closed database connection approximates a non-useable
	// database connection
	InvalidDB, err = sqlx.Connect("postgres", config.ConnStr)
	Expect(err).NotTo(HaveOccurred())
	InvalidDB.Close()

	db, err = sqlx.Connect("postgres", config.ConnStr)
	Expect(err).NotTo(HaveOccurred())

	db.MustExec(drop_clusters_ddl)
	db.MustExec(drop_cluster_test_schema)

	db.MustExec(cluster_test_schema)
	db.MustExec(clusters_ddl)
	db.MustExec(cluster_test_searchpath)
})

var _ = AfterSuite(func() {
	db.Close()
})

var _ = Describe("Cluster", func() {

	var (
		cluster         *models.Cluster
		cluster1        *models.Cluster
		cluster2        *models.Cluster
		requestId       string
		notacluster     *models.Cluster
		clusters        []models.Cluster
		err             error
		dao             ClusterDao
		tx              *sqlx.Tx
		terraformConfig []byte
	)

	BeforeEach(func() {
		dao = ClusterDao{}

		requestId = "c12c2d58-2af0-11e8-b467-0ed5f89f718b"

		// Create a fresh transaction for each test
		tx, err = db.Beginx()
		Expect(err).NotTo(HaveOccurred())

		outputs := map[string]models.Output{
			"foobar": models.Output{
				Sensitive: "true",
				Type:      "foo",
				Value:     "bar",
			},
		}
		js, err := json.Marshal(outputs)
		Expect(err).NotTo(HaveOccurred())

		cluster1 = &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_1", Status: "provisioned", Message: "", TerraformState: nil, Outputs: js}
		cluster2 = &models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_2", Status: "provisioned", Message: "", TerraformState: nil, Outputs: js}
		notacluster = &models.Cluster{Id: "a19e1bfe-0ec5-11ea-ba89-0ed0f89f718b", Name: "notacluster", Status: "nothere", Message: "", TerraformState: nil, Outputs: nil}
		terraformConfig = []byte(`{"provider":{"google":{}}}`)
	})

	AfterEach(func() {
		// Ensure the test data is removed
		db.MustExec(truncate_clusters)
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
				cluster, err = dao.CreateCluster(db, terraformConfig, requestId)
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
			It("Should be in the requested status", func() {
				Expect(cluster.Status).To(Equal(models.ClusterStatusRequested))
			})
			It("Should have written the config in the config field", func() {
				Expect(cluster.TerraformConfig).To(Equal(terraformConfig))
			})
			It("Should use the request id for the cluster id", func() {
				Expect(cluster.Id).To(Equal(requestId))
			})
		})

		Context("Without terraform configuration", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(db, nil, requestId)
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
				cluster, err = dao.CreateCluster(db, terraformConfig, "")
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
				cluster, err = dao.CreateCluster(InvalidDB, nil, requestId)
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
				db.MustExec("INSERT INTO clusters (id,name,status,message,outputs,terraform_state) VALUES ($1,$2,$3,$4,$5,$6)", cluster1.Id, cluster1.Name, cluster1.Status, cluster1.Message, cluster1.Outputs, cluster1.TerraformState)
				cluster, err = dao.GetCluster(db, cluster1.Id, requestId)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should return the expected cluster", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
			})
		})

		Context("When the cluster does not exist", func() {
			BeforeEach(func() {
				// Without inserting any clusters into database
				cluster, err = dao.GetCluster(db, cluster1.Id, requestId)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("When a clusterId was not specified", func() {
			BeforeEach(func() {
				cluster, err = dao.GetCluster(db, "", requestId)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("When a requestId was not specified", func() {
			BeforeEach(func() {
				cluster, err = dao.GetCluster(db, cluster1.Id, "")
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
				cluster, err = dao.GetCluster(InvalidDB, cluster1.Id, requestId)
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
				db.MustExec("INSERT INTO clusters (id,name,status,message,terraform_state) VALUES ($1,$2,$3,$4,$5)", cluster1.Id, cluster1.Name, cluster1.Status, cluster1.Message, cluster1.TerraformState)
				db.MustExec("INSERT INTO clusters (id,name,status,message,terraform_state) VALUES ($1,$2,$3,$4,$5)", cluster2.Id, cluster2.Name, cluster2.Status, cluster2.Message, cluster2.TerraformState)
				clusters, err = dao.GetClusters(db, requestId)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return clusters", func() {
				Expect(clusters).To(HaveLen(2))
			})
			It("Should return the expected clusters", func() {
				Expect(clusters).To(HaveLen(2))
				Expect(clusters[0].Id).To(Equal(cluster1.Id))
				Expect(clusters[1].Id).To(Equal(cluster2.Id))
			})
		})

		Context("When no clusters exist", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(db, requestId)
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return an empty list of Clusters", func() {
				Expect(clusters).To(HaveLen(0))
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

	Describe("Updating a cluster", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				db.MustExec("INSERT INTO clusters (id,name,status,message,terraform_state) VALUES ($1,$2,$3,$4,$5)", cluster1.Id, cluster1.Name, cluster1.Status, cluster1.Message, cluster1.TerraformState)
				updated_cluster := &models.Cluster{
					Id:     cluster1.Id,
					Status: "different_status",
					Name:   cluster1.Name,
				}
				cluster, err = dao.UpdateCluster(db, updated_cluster, requestId)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should return the expected cluster", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
			})
			It("Should have been updated for the cluster returned", func() {
				Expect(cluster.Status).To(Equal("different_status"))
			})
			It("Should have been updated for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Status).To(Equal("different_status"))
			})
		})

		Context("When nothing is different", func() {
			BeforeEach(func() {
				db.MustExec("INSERT INTO clusters (id,name,status,message,terraform_state) VALUES ($1,$2,$3,$4,$5)", cluster1.Id, cluster1.Name, cluster1.Status, cluster1.Message, cluster1.TerraformState)
				cluster, err = dao.UpdateCluster(db, cluster1, requestId)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should return the expected cluster", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
			})
			It("Should have no changes for the cluster returned", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
				Expect(cluster.Name).To(Equal(cluster1.Name))
				Expect(cluster.Status).To(Equal(cluster1.Status))
			})
			It("Should have no changes for the cluster saved", func() {
				// In order to use sqlx scanning, cluster needs to be empty struct
				cluster := models.Cluster{}
				err := db.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", cluster1.Id)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Id).To(Equal(cluster1.Id))
				Expect(cluster.Name).To(Equal(cluster1.Name))
				Expect(cluster.Status).To(Equal(cluster1.Status))
			})
		})

		Context("When the cluster does not exist", func() {
			BeforeEach(func() {
				cluster, err = dao.UpdateCluster(db, notacluster, requestId)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
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

	Describe("Deleting clusters", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				db.MustExec("INSERT INTO clusters (id,name,status,message,terraform_state) VALUES ($1,$2,$3,$4,$5)", cluster1.Id, cluster1.Name, cluster1.Status, cluster1.Message, cluster1.TerraformState)
				cluster, err = dao.DeleteCluster(db, cluster1.Id, requestId)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should return the expected cluster", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
			})
			It("The returned cluster should be getting destroyed", func() {
				Expect(cluster.Status).To(Equal("destroying"))
			})
		})

		Context("When the cluster does not exist", func() {
			BeforeEach(func() {
				cluster, err = dao.DeleteCluster(db, cluster1.Id, requestId)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("When the cluster has already been destroyed", func() {
			BeforeEach(func() {
				cluster1.Status = "destroyed"
				rows, insert_err := db.NamedQuery(`INSERT INTO clusters (id,name,status,message) VALUES (:id,:name,:status,:message) RETURNING id`, cluster1)
				Expect(insert_err).NotTo(HaveOccurred())
				var id string
				if rows.Next() {
					rows.Scan(&id)
				}
				cluster, err = dao.DeleteCluster(db, id, requestId)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("When the cluster is already being destroyed (destroying)", func() {
			BeforeEach(func() {
				cluster1.Status = "destroying"
				rows, insert_err := db.NamedQuery(`INSERT INTO clusters (id,name,status,message) VALUES (:id,:name,:status,:message) RETURNING id`, cluster1)
				Expect(insert_err).NotTo(HaveOccurred())
				var id string
				if rows.Next() {
					rows.Scan(&id)
				}
				cluster, err = dao.DeleteCluster(db, id, requestId)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should return the expected cluster", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
			})
			It("The returned cluster should be getting destroyed", func() {
				Expect(cluster.Status).To(Equal("destroying"))
			})
		})

	})

})
