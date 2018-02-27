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
	config                   app.ServerConfig
	cluster_test_schema      = `CREATE SCHEMA IF NOT EXISTS cluster_test`
	drop_cluster_test_schema = `DROP SCHEMA IF EXISTS cluster_test CASCADE`
	cluster_test_searchpath  = `ALTER ROLE taos SET search_path TO cluster_test, public`
	clusters_ddl             = `
		CREATE TABLE IF NOT EXISTS cluster_test.clusters (
				id 			UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
				name 		text,
				status 	text
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
		cluster     *models.Cluster
		cluster1    *models.Cluster
		cluster2    *models.Cluster
		notacluster *models.Cluster
		clusters    []models.Cluster
		clusterId   string
		err         error
		dao         ClusterDao
		tx          *sqlx.Tx
	)

	BeforeEach(func() {
		dao = ClusterDao{}

		// Create a fresh transaction for each test
		tx, err = db.Beginx()
		Expect(err).NotTo(HaveOccurred())

		cluster1 = &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_1", Status: "provisioned"}
		cluster2 = &models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_2", Status: "provisioned"}
		notacluster = &models.Cluster{Id: "a19e1bfe-0ec5-11ea-ba89-0ed0f89f718b", Name: "notacluster", Status: "nothere"}
	})

	AfterEach(func() {
		// Ensure the test data is removed
		db.MustExec(truncate_clusters)

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

	Describe("Setting Cluster fields", func() {
		Context("When changing the cluster status for an existing cluster", func() {
			BeforeEach(func() {
				rows, err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(err).NotTo(HaveOccurred())
				if rows.Next() {
					rows.Scan(&clusterId)
				}
				new_cluster := &models.Cluster{
					Id:     clusterId,
					Status: "different_status",
					Name:   cluster1.Name,
				}
				cluster, err = dao.UpdateCluster(db, new_cluster)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should change status", func() {
				updated_cluster := models.Cluster{}
				err := db.Get(&updated_cluster, "SELECT * FROM clusters WHERE id=$1", clusterId)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated_cluster.Status).To(Equal("different_status"))
			})
		})

		Context("When changing the cluster status for a non-existing cluster", func() {
			BeforeEach(func() {
				cluster, err = dao.UpdateCluster(db, notacluster)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
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
		Context("When deleting an existing cluster", func() {
			BeforeEach(func() {
				rows, err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(err).NotTo(HaveOccurred())
				if rows.Next() {
					rows.Scan(&clusterId)
				}
				cluster, err = dao.DeleteCluster(db, clusterId)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the same cluster", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
			})
			It("The returned cluster should have a deleting status", func() {
				Expect(cluster.Status).To(Equal("deleting"))
			})
		})

		Context("When deleting a cluster that does not exist", func() {
			BeforeEach(func() {
				cluster, err = dao.DeleteCluster(db, cluster1.Id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should return a nil cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("When deleting a cluster that has already been deleted", func() {
			BeforeEach(func() {
				cluster1.Status = "deleted"
				rows, insert_err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(insert_err).NotTo(HaveOccurred())
				if rows.Next() {
					rows.Scan(&clusterId)
				}
				cluster, err = dao.DeleteCluster(db, clusterId)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should return a nil cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("When deleting a cluster that is already deleting", func() {
			BeforeEach(func() {
				cluster1.Status = "deleting"
				rows, insert_err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(insert_err).NotTo(HaveOccurred())
				if rows.Next() {
					rows.Scan(&clusterId)
				}
				cluster, err = dao.DeleteCluster(db, clusterId)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should return a nil cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("When attempting to delete with an invalid ID", func() {
			BeforeEach(func() {
				cluster, err = dao.DeleteCluster(db, "invalid-uuid")
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should return a nil cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})
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
		Context("When a cluster is successfully created", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(db)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should be in the provisioning status", func() {
				Expect(cluster.Status).To(Equal("provisioning"))
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

	Describe("Retrieving a Cluster", func() {
		Context("With an id that exists", func() {
			BeforeEach(func() {
				rows, err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(err).NotTo(HaveOccurred())
				if rows.Next() {
					rows.Scan(&clusterId)
				}
				cluster, err = dao.GetCluster(db, clusterId)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster of the same id", func() {
				Expect(cluster).To(Equal(cluster1))
			})
		})

		Context("That does not exist", func() {
			BeforeEach(func() {
				cluster1, err = dao.GetCluster(db, "596abbac-0ed1-11e8-ba89-0ed5f89f718b")
			})
			It("Should return a nil cluster", func() {
				Expect(cluster1).Should(BeNil())
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
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
		Context("When clusters exist", func() {
			BeforeEach(func() {
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster2.Id, cluster2.Name, cluster2.Status)
				clusters, err = dao.GetClusters(db)
			})
			It("Should return clusters", func() {
				Expect(clusters).To(HaveLen(2))
			})
		})

		Context("When clusters do not exist", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(db)
			})
			It("Should return an empty list of Clusters", func() {
				Expect(clusters).To(HaveLen(0))
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

	})

})
