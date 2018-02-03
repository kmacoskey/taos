package services_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	. "github.com/kmacoskey/taos/services"
)

var _ = Describe("Cluster", func() {

	var (
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

		cluster1 = &models.Cluster{Id: 1, Name: "cluster", Status: "status"}
		cluster2 = &models.Cluster{Id: 2, Name: "cluster", Status: "status"}
	})

	Describe("Retrieving a Cluster for a specific id", func() {
		Context("A cluster is returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewValidClusterDao())
			})
			It("Should return a cluster of the same id", func() {
				Expect(cs.GetCluster(rc, 1)).To(Equal(cluster1))
			})
		})

		Context("A cluster is not returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewEmptyClusterDao())
				cluster1, err = cs.GetCluster(rc, 1)
			})
			It("Should return an empty Cluster", func() {
				Expect(cluster1).To(Equal(&models.Cluster{}))
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
		})
	})

	Describe("Retrieving all clusters", func() {
		Context("When Clusters are returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewValidClusterDao())
			})
			It("Should return a slice of all clusters", func() {
				Expect(cs.GetClusters(rc)).To(HaveLen(2))
			})
		})

		Context("When no Clusters are returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewEmptyClusterDao())
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

type ValidClusterDao struct{}

func NewValidClusterDao() *ValidClusterDao {
	return &ValidClusterDao{}
}

func (dao *ValidClusterDao) GetCluster(rc app.RequestContext, id int) (*models.Cluster, error) {
	return &models.Cluster{Id: 1, Name: "cluster", Status: "status"}, nil
}

func (dao *ValidClusterDao) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
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

func (dao *EmptyClusterDao) GetCluster(rc app.RequestContext, id int) (*models.Cluster, error) {
	return &models.Cluster{}, errors.New("foo")
}

func (dao *EmptyClusterDao) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	return clusters, nil
}
