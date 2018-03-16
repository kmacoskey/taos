package daos_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
				terraform_config 	json
		)`
	truncate_clusters = `TRUNCATE TABLE clusters`
	drop_clusters_ddl = `DROP TABLE IF EXISTS cluster_test.clusters CASCADE`
	create_pgcrypto   = `CREATE EXTENSION pgcrypto`
)

var _ = BeforeSuite(func() {
	err := app.LoadServerConfig(&config, "../")
	Expect(err).NotTo(HaveOccurred())

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
		notacluster     *models.Cluster
		clusters        []models.Cluster
		err             error
		dao             ClusterDao
		tx              *sqlx.Tx
		terraformConfig []byte
	)

	BeforeEach(func() {
		dao = ClusterDao{}

		// Create a fresh transaction for each test
		tx, err = db.Beginx()
		Expect(err).NotTo(HaveOccurred())

		cluster1 = &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_1", Status: "provisioned"}
		cluster2 = &models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_2", Status: "provisioned"}
		notacluster = &models.Cluster{Id: "a19e1bfe-0ec5-11ea-ba89-0ed0f89f718b", Name: "notacluster", Status: "nothere"}
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
				cluster, err = dao.CreateCluster(db, terraformConfig)
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
		})

		Context("Without terraform configuration", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(db, nil)
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
				cluster, err = dao.CreateCluster(InvalidDB, nil)
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
				rows, err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(err).NotTo(HaveOccurred())
				var cluster_id string
				if rows.Next() {
					rows.Scan(&cluster_id)
				}
				cluster, err = dao.GetCluster(db, cluster_id)
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
				cluster, err = dao.GetCluster(db, cluster1.Id)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

		Context("When an id was not specified", func() {
			BeforeEach(func() {
				cluster, err = dao.GetCluster(db, "")
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
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster2.Id, cluster2.Name, cluster2.Status)
				clusters, err = dao.GetClusters(db)
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
				clusters, err = dao.GetClusters(db)
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
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
				updated_cluster := &models.Cluster{
					Id:     cluster1.Id,
					Status: "different_status",
					Name:   cluster1.Name,
				}
				cluster, err = dao.UpdateCluster(db, updated_cluster)
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
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
				cluster, err = dao.UpdateCluster(db, cluster1)
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
				cluster, err = dao.UpdateCluster(db, notacluster)
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
				rows, err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(err).NotTo(HaveOccurred())
				var id string
				if rows.Next() {
					rows.Scan(&id)
				}
				cluster, err = dao.DeleteCluster(db, id)
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
				cluster, err = dao.DeleteCluster(db, cluster1.Id)
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
				rows, insert_err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(insert_err).NotTo(HaveOccurred())
				var id string
				if rows.Next() {
					rows.Scan(&id)
				}
				cluster, err = dao.DeleteCluster(db, id)
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
				rows, insert_err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(insert_err).NotTo(HaveOccurred())
				var id string
				if rows.Next() {
					rows.Scan(&id)
				}
				cluster, err = dao.DeleteCluster(db, id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).Should(BeNil())
			})
		})

	})

})
