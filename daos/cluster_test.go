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
	cluster_test_searchpath  = `SET search_path TO cluster_test`
	clusters_ddl             = `
		CREATE TABLE IF NOT EXISTS cluster_test.clusters (
				id 			integer,
				name 		text,
				status 	text
		)`
	truncate_clusters = `TRUNCATE TABLE clusters`
	drop_clusters_ddl = `DROP TABLE IF EXISTS cluster_test.clusters CASCADE`
)

var _ = BeforeSuite(func() {
	err := app.LoadServerConfig(&config, "../")
	Expect(err).NotTo(HaveOccurred())

	db, err = sqlx.Connect("postgres", config.ConnStr)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	db.Close()
})

var _ = Describe("Cluster", func() {

	var (
		cluster1 *models.Cluster
		cluster2 *models.Cluster
		clusters []models.Cluster
		err      error
		dao      ClusterDao
		rc       app.RequestContext
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

		// Before each test, ensure the search path
		// is set to find the correct testing tables
		tx.MustExec(cluster_test_searchpath)

		db.MustExec(cluster_test_schema)
		db.MustExec(clusters_ddl)

		cluster1 = &models.Cluster{Id: 1, Name: "cluster_1", Status: "provisioned"}
		cluster2 = &models.Cluster{Id: 2, Name: "cluster_2", Status: "provisioned"}
	})

	AfterEach(func() {
		rc.Tx().Commit()
	})

	Describe("Retrieving clusters", func() {
		Context("When clusters exist", func() {
			BeforeEach(func() {
				rc.Tx().MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
				rc.Tx().MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster2.Id, cluster2.Name, cluster2.Status)
			})
			AfterEach(func() {
				rc.Tx().MustExec(truncate_clusters)
			})
			It("Should return clusters", func() {
				Expect(dao.GetClusters(rc)).To(HaveLen(2))
			})
		})

		Context("When clusters do not exist", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(rc)
			})
			It("Should return an empty list of Clusters", func() {
				Expect(dao.GetClusters(rc)).To(HaveLen(0))
			})
			It("Should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("When the database relation does not exist", func() {
			BeforeEach(func() {
				rc.Tx().MustExec(drop_clusters_ddl)
				clusters, err = dao.GetClusters(rc)
			})
			It("Should return an error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should return a nil list of clusters", func() {
				Expect(clusters).Should(BeNil())
			})
		})

	})

	Describe("Retrieving a Cluster", func() {
		Context("With an id that exists", func() {
			JustBeforeEach(func() {
				rc.Tx().MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
			})
			AfterEach(func() {
				rc.Tx().MustExec(truncate_clusters)
			})
			It("Should return a cluster of the same id", func() {
				Expect(dao.GetCluster(rc, 1)).To(Equal(cluster1))
			})
		})

		Context("That does not exist", func() {
			BeforeEach(func() {
				cluster1, err = dao.GetCluster(rc, 1)
			})
			It("Should return a nil cluster", func() {
				Expect(cluster1).Should(BeNil())
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("When the database relation does not exist", func() {
			BeforeEach(func() {
				rc.Tx().MustExec(drop_clusters_ddl)
				cluster1, err = dao.GetCluster(rc, 1)
			})
			It("Should return an error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should return a nil cluster", func() {
				Expect(cluster1).Should(BeNil())
			})
		})

	})

})
