package services_test

import (
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	. "github.com/kmacoskey/taos/services"
)

// Structure to hold test clusters for DAO interfaces defined outside the spec
//  This essentially facilitates a datastore to handle GET/UPDATE actions
var clustersMap map[string]*models.Cluster

var _ = Describe("Cluster", func() {

	var (
		cluster  *models.Cluster
		cluster1 *models.Cluster
		cluster2 *models.Cluster
		clusters []models.Cluster
		cs       *ClusterService
		rc       app.RequestContext
		err      error
	)

	BeforeEach(func() {
		// Create a new RequestContext for each test
		rc = app.RequestContext{}

		cluster1 = &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}
		cluster2 = &models.Cluster{Id: "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}

		clustersMap = make(map[string]*models.Cluster)
		clustersMap["a19e2758-0ec5-11e8-ba89-0ed5f89f718b"] = cluster1
		clustersMap["a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b"] = cluster2
	})

	// ======================================================================
	//                      _
	//   ___ _ __ ___  __ _| |_ ___
	//  / __| '__/ _ \/ _` | __/ _ \
	// | (__| | |  __/ (_| | ||  __/
	//  \___|_|  \___|\__,_|\__\___|
	//
	// ======================================================================

	Describe("Creating a Cluster", func() {
		Context("A cluster is returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewValidClusterDao(), NewMockDB().db)
				cluster, err = cs.CreateCluster(rc)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should have a cluster returned with status provisioning", func() {
				Expect(cluster.Status).To(Equal("provisioning"))
			})
			It("Should set the cluster status in the daos", func() {
				Expect(cluster.Status).To(Equal("provisioning"))
			})
			It("Should eventually be provisioned", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Status
				}, 2, 0.5).Should(Equal("provisioned"))
			})
		})

		Context("A cluster is not returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewEmptyClusterDao(), NewMockDB().db)
				cluster, err = cs.CreateCluster(rc)
			})
			It("Should return an empty Cluster", func() {
				Expect(cluster).To(Equal(&models.Cluster{}))
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
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

	Describe("Retrieving a Cluster for a specific id", func() {
		Context("A cluster is returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewValidClusterDao(), NewMockDB().db)
			})
			It("Should return a cluster of the same id", func() {
				Expect(cs.GetCluster(rc, "a19e2758-0ec5-11e8-ba89-0ed5f89f718b")).To(Equal(cluster1))
			})
		})

		Context("A cluster is not returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewEmptyClusterDao(), NewMockDB().db)
				cluster1, err = cs.GetCluster(rc, "a19e2758-0ec5-11e8-ba89-0ed5f89f718b")
			})
			It("Should return an empty Cluster", func() {
				Expect(cluster1).To(Equal(&models.Cluster{}))
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

	Describe("Retrieving all clusters", func() {
		Context("When Clusters are returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewValidClusterDao(), NewMockDB().db)
			})
			It("Should return a slice of all clusters", func() {
				Expect(cs.GetClusters(rc)).To(HaveLen(2))
			})
		})

		Context("When no Clusters are returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewEmptyClusterDao(), NewMockDB().db)
				clusters, err = cs.GetClusters(rc)
			})
			It("Should return an empty list of Clusters", func() {
				Expect(clusters).To(HaveLen(0))
			})
			It("should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

		})
	})

})

func NewMockDB() *MockDB {
	return &MockDB{}
}

type MockDB struct {
	db *sqlx.DB
}

type ValidClusterDao struct{}

func NewValidClusterDao() *ValidClusterDao {
	return &ValidClusterDao{}
}

func (dao *ValidClusterDao) CreateCluster(db *sqlx.DB) (*models.Cluster, error) {
	return &models.Cluster{Id: "a19e2758-0ec5-11e8-ba89-0ed5f89f718b", Name: "cluster", Status: "status"}, nil
}

func (dao *ValidClusterDao) UpdateCluster(db *sqlx.DB, cluster *models.Cluster) (*models.Cluster, error) {
	clustersMap[cluster.Id] = cluster
	return clustersMap[cluster.Id], nil
}

func (dao *ValidClusterDao) GetCluster(db *sqlx.DB, id string) (*models.Cluster, error) {
	return clustersMap[id], nil
}

func (dao *ValidClusterDao) GetClusters(db *sqlx.DB) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	cluster := models.Cluster{}
	clusters = append(clusters, cluster)
	clusters = append(clusters, cluster)
	return clusters, nil
}

type EmptyClusterDao struct{}

func NewEmptyClusterDao() *EmptyClusterDao {
	return &EmptyClusterDao{}
}

func (dao *EmptyClusterDao) CreateCluster(db *sqlx.DB) (*models.Cluster, error) {
	return &models.Cluster{}, errors.New("foo")
}

func (dao *EmptyClusterDao) UpdateCluster(db *sqlx.DB, cluster *models.Cluster) (*models.Cluster, error) {
	return cluster, nil
}

func (dao *EmptyClusterDao) GetCluster(db *sqlx.DB, id string) (*models.Cluster, error) {
	return &models.Cluster{}, errors.New("foo")
}

func (dao *EmptyClusterDao) GetClusters(db *sqlx.DB) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	return clusters, nil
}
