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
		rc          app.RequestContext
	)

	BeforeEach(func() {
		dao = ClusterDao{}

		// Including a fresh transaction for each test
		tx, err := db.Beginx()
		Expect(err).NotTo(HaveOccurred())

		// Create a new RequestContext for each test
		rc = app.RequestContext{}
		// Set that fresh transaction in the request context
		rc.SetTx(tx)

		cluster1 = &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_1", Status: "provisioned"}
		cluster2 = &models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster_2", Status: "provisioned"}
		notacluster = &models.Cluster{Id: "a19e1bfe-0ec5-11ea-ba89-0ed0f89f718b", Name: "notacluster", Status: "nothere"}
	})

	AfterEach(func() {
		// Ensure the test data is removed
		db.MustExec(truncate_clusters)
	})

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
				cluster, err = dao.UpdateCluster(rc, new_cluster)
				// Must commit the transaction in order to test that it completed
				rc.Tx().Commit()
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
				cluster, err = dao.UpdateCluster(rc, notacluster)
				rc.Tx().Commit()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

	})

	Describe("Creating cluster", func() {
		Context("When a cluster is successfully created", func() {
			BeforeEach(func() {
				cluster, err = dao.CreateCluster(rc)
				rc.Tx().Commit()
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

	Describe("Retrieving a Cluster", func() {
		Context("With an id that exists", func() {
			BeforeEach(func() {
				rows, err := db.NamedQuery(`INSERT INTO clusters (id,name,status) VALUES (:id,:name,:status) RETURNING id`, cluster1)
				Expect(err).NotTo(HaveOccurred())
				if rows.Next() {
					rows.Scan(&clusterId)
				}
				cluster, err = dao.GetCluster(rc, clusterId)
				rc.Tx().Commit()
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
				cluster1, err = dao.GetCluster(rc, "596abbac-0ed1-11e8-ba89-0ed5f89f718b")
				rc.Tx().Commit()
			})
			It("Should return a nil cluster", func() {
				Expect(cluster1).Should(BeNil())
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
		})

	})

	Describe("Retrieving clusters", func() {
		Context("When clusters exist", func() {
			BeforeEach(func() {
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
				db.MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster2.Id, cluster2.Name, cluster2.Status)
				clusters, err = dao.GetClusters(rc)
				rc.Tx().Commit()
			})
			It("Should return clusters", func() {
				Expect(clusters).To(HaveLen(2))
			})
		})

		Context("When clusters do not exist", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(rc)
				rc.Tx().Commit()
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
