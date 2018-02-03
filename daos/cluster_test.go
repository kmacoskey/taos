package daos_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	. "github.com/kmacoskey/taos/daos"
	"github.com/kmacoskey/taos/models"
)

var db *sqlx.DB
var config app.ServerConfig
var schema = `CREATE SCHEMA IF NOT EXISTS cluster_test`
var searchpath = `SET search_path TO cluster_test`
var ddl = `
CREATE TABLE IF NOT EXISTS cluster_test.clusters (
    id 			integer,
    name 		text,
    status 	text
)
`
var drop_schema = `DROP SCHEMA IF EXISTS cluster_test CASCADE`
var drop_ddl = `DROP TABLE IF EXISTS cluster_test.cluster CASCADE`
var truncate = `TRUNCATE TABLE clusters`

// A global is used to hold the request context
//  but it is created anew for every test/transaction
var rc app.RequestContext

var _ = BeforeSuite(func() {
	err := app.LoadServerConfig(&config, "../")
	Expect(err).NotTo(HaveOccurred())

	db, err = sqlx.Connect("postgres", config.ConnStr)
	Expect(err).NotTo(HaveOccurred())

	db.MustExec(schema)
	db.MustExec(ddl)
})

var _ = AfterSuite(func() {
	db.MustExec(drop_ddl)
	db.MustExec(drop_schema)

	db.Close()
})

var _ = Describe("Cluster", func() {

	var (
		cluster1 *models.Cluster
		cluster2 *models.Cluster
		clusters []models.Cluster
		err      error
		dao      ClusterDao
	)

	BeforeEach(func() {
		dao = ClusterDao{}

		// Create a new RequestContext for each test
		rc = app.RequestContext{}

		// Including a fresh transaction for each test
		tx, err := db.Beginx()
		Expect(err).NotTo(HaveOccurred())

		rc.SetTx(tx)

		tx.MustExec(searchpath)

		cluster1 = &models.Cluster{Id: 1, Name: "cluster_1", Status: "provisioned"}
		cluster2 = &models.Cluster{Id: 2, Name: "cluster_2", Status: "provisioned"}
	})

	AfterEach(func() {
		// After each test, truncate the test table
		// and commit the transaction to cleanup
		rc.Tx().MustExec(truncate)
		rc.Tx().Commit()
	})

	Describe("Retrieving all clusters", func() {
		Context("When clusters exist", func() {
			BeforeEach(func() {
				rc.Tx().MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
				rc.Tx().MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster2.Id, cluster2.Name, cluster2.Status)
			})
			It("Should return all clusters", func() {
				Expect(dao.GetClusters(rc)).To(HaveLen(2))
			})
		})

		Context("When no clusters exist", func() {
			BeforeEach(func() {
				clusters, err = dao.GetClusters(rc)
			})
			It("Should return an empty list of Clusters", func() {
				Expect(dao.GetClusters(rc)).To(HaveLen(0))
			})
			It("should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

		})
	})

	Describe("Retrieving a Cluster", func() {
		Context("With an id of 1", func() {
			BeforeEach(func() {
				rc.Tx().MustExec("INSERT INTO clusters (id, name, status) VALUES ($1, $2, $3)", cluster1.Id, cluster1.Name, cluster1.Status)
			})
			It("Should return a cluster of the same id", func() {
				Expect(dao.GetCluster(rc, 1)).To(Equal(cluster1))
			})
		})

		Context("That does not exist", func() {
			BeforeEach(func() {
				cluster1, err = dao.GetCluster(rc, 1)
			})
			It("Should return nil", func() {
				Expect(cluster1).Should(BeNil())
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
		})
	})

})
