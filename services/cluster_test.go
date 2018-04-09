package services_test

import (
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"errors"

	"github.com/satori/go.uuid"

	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	. "github.com/kmacoskey/taos/services"
	"github.com/kmacoskey/taos/terraform"
)

var _ = Describe("Cluster", func() {

	var (
		cluster                       *models.Cluster
		cluster1UUID                  string
		cluster1                      *models.Cluster
		cluster2UUID                  string
		cluster2                      *models.Cluster
		clusters                      []models.Cluster
		cs                            *ClusterService
		rc                            app.RequestContext
		err                           error
		validTerraformConfig          []byte
		invalidTerraformConfig        []byte
		validNoOutputsTerraformConfig []byte
	)

	BeforeEach(func() {
		log.SetLevel(log.FatalLevel)

		// Create a new RequestContext for each test
		rc = app.RequestContext{}

		validTerraformConfig = []byte(`{"provider":{"google":{"project":"data-gp-toolsmiths","region":"us-central1"}},"output":{"foo":{"value":"bar"}}}`)
		validNoOutputsTerraformConfig = []byte(`{"provider":{"google":{"project":"data-gp-toolsmiths","region":"us-central1"}}}`)
		invalidTerraformConfig = []byte(`notjson`)

		cluster1UUID = "a19e2758-0ec5-11e8-ba89-0ed5f89f718b"
		cluster1 = &models.Cluster{
			Id:              cluster1UUID,
			Name:            "cluster",
			Status:          "status",
			TerraformConfig: []byte(`{"provider":{"google":{}}}`),
		}

		cluster2UUID = "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b"
		cluster2 = &models.Cluster{
			Id:              cluster2UUID,
			Name:            "cluster",
			Status:          "status",
			TerraformConfig: []byte(`{"provider":{"google":{}}}`),
		}
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
				clustersMap := make(map[string]*models.Cluster)
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				rc.SetTerraformConfig(validTerraformConfig)
				cluster, err = cs.CreateCluster(rc)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			FIt("Should eventually be provisioned", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Status
				}, 5, 0.5).Should(Equal("provision_success"))
			})
			It("Should eventually set the message to the Terraform client output", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Message
				}, 5, 0.5).Should(ContainSubstring("Apply complete! Resources: 0 added, 0 changed, 0 destroyed."))
			})
			It("Should eventually set the Terraform state", func() {
				Eventually(func() []byte {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.TerraformState
				}, 5, 0.5).ShouldNot(BeNil())
			})
			It("Should eventually set the Terraform outputs", func() {
				Eventually(func() []byte {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Outputs
				}, 5, 0.5).ShouldNot(BeNil())
			})
		})

		Context("When a cluster is not returned from the dao", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewEmptyClusterDao(), NewMockDB().db)
				cluster, err = cs.CreateCluster(rc)
			})
			It("Should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("When invalid terraform config is used", func() {
			BeforeEach(func() {
				clustersMap := make(map[string]*models.Cluster)
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				rc.SetTerraformConfig(invalidTerraformConfig)
				cluster, err = cs.CreateCluster(rc)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should eventually change status to reflect an error", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Status
				}, 2, 0.5).Should(Equal("provision_failed"))
			})
			It("Should eventually set the message to the terraform client output", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Message
				}, 5, 0.5).Should(ContainSubstring(terraform.ErrorInvalidConfig))
			})
		})

		Context("When there are no outputs defined in the config", func() {
			BeforeEach(func() {
				clustersMap := make(map[string]*models.Cluster)
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				rc.SetTerraformConfig(validNoOutputsTerraformConfig)
				cluster, err = cs.CreateCluster(rc)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a cluster", func() {
				Expect(cluster).NotTo(BeNil())
			})
			It("Should eventually be provisioned", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Status
				}, 5, 0.5).Should(Equal("provision_success"))
			})
			It("Should eventually set the message to the Terraform client output", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Message
				}, 5, 0.5).Should(ContainSubstring("Apply complete! Resources: 0 added, 0 changed, 0 destroyed."))
			})
			It("Should eventually set the Terraform state", func() {
				Eventually(func() []byte {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.TerraformState
				}, 5, 0.5).ShouldNot(BeNil())
			})
			It("Should eventually set the Terraform to nil", func() {
				Eventually(func() []byte {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Outputs
				}, 5, 0.5).Should(BeNil())
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

	Describe("Getting a cluster", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				clustersMap := make(map[string]*models.Cluster)
				clustersMap[cluster1.Id] = cluster1
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				cluster, err = cs.GetCluster(rc, cluster1.Id)
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
				cs = NewClusterService(NewEmptyClusterDao(), NewMockDB().db)
				cluster, err = cs.GetCluster(rc, cluster1.Id)
			})
			It("Should error", func() {
				Expect(err).Should(HaveOccurred())
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

	Describe("Getting all clusters", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				clustersMap := make(map[string]*models.Cluster)
				clustersMap[cluster1.Id] = cluster1
				clustersMap[cluster2.Id] = cluster2
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				clusters, err = cs.GetClusters(rc)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return clusters", func() {
				Expect(clusters).To(HaveLen(2))
			})
			It("Should return the expected clusters", func() {
				Expect(clusters).To(ContainElement(*cluster1))
				Expect(clusters).To(ContainElement(*cluster2))
			})
		})

		Context("When there are no clusters", func() {
			BeforeEach(func() {
				cs = NewClusterService(NewEmptyClusterDao(), NewMockDB().db)
				clusters, err = cs.GetClusters(rc)
			})
			It("should not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should return an empty list of Clusters", func() {
				Expect(clusters).To(HaveLen(0))
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
				clustersMap := make(map[string]*models.Cluster)
				clustersMap[cluster1UUID] = cluster1
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				cluster, err = cs.DeleteCluster(rc, cluster1UUID)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the expected cluster", func() {
				Expect(cluster.Id).To(Equal(cluster1.Id))
			})
			It("The should be destroying", func() {
				Expect(cluster.Status).To(Equal("destroying"))
			})
			It("Should eventually be destroyed", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Status
				}, 3, 0.5).Should(Equal("destroyed"))
			})
			It("Should eventually set the message to the terraform client output", func() {
				Eventually(func() string {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.Message
				}, 5, 0.5).Should(ContainSubstring("Destroy complete! Resources: 0 destroyed."))
			})
			It("Should eventually set the Terraform state", func() {
				Eventually(func() []byte {
					c, err := cs.GetCluster(rc, cluster.Id)
					Expect(err).NotTo(HaveOccurred())
					return c.TerraformState
				}, 5, 0.5).ShouldNot(BeNil())
			})
		})

		Context("When it does not exist", func() {
			BeforeEach(func() {
				clustersMap := make(map[string]*models.Cluster)
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				cluster, err = cs.DeleteCluster(rc, cluster1.Id)
			})
			It("should error", func() {
				Expect(err).Should(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
			})
		})

		Context("That has already been deleted", func() {
			BeforeEach(func() {
				clustersMap := make(map[string]*models.Cluster)
				cluster1.Status = "destroyed"
				clustersMap[cluster1.Id] = cluster1
				cs = NewClusterService(NewValidClusterDao(clustersMap), NewMockDB().db)
				cluster, err = cs.DeleteCluster(rc, cluster1.Id)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return a cluster", func() {
				Expect(cluster).To(BeNil())
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

type ValidClusterDao struct {
	clustersMap map[string]*models.Cluster
}

func NewValidClusterDao(cm map[string]*models.Cluster) *ValidClusterDao {
	return &ValidClusterDao{
		clustersMap: cm,
	}
}

func (dao *ValidClusterDao) CreateCluster(db *sqlx.DB, config []byte, requestId string) (*models.Cluster, error) {
	uuid := uuid.Must(uuid.NewV4()).String()
	dao.clustersMap[uuid] = &models.Cluster{
		Id:              uuid,
		Name:            "cluster",
		Status:          "status",
		TerraformConfig: config,
	}
	return dao.clustersMap[uuid], nil
}

func (dao *ValidClusterDao) UpdateClusterField(db *sqlx.DB, id string, field string, value interface{}, requestId string) error {
	cluster := &models.Cluster{}
	cluster = dao.clustersMap[id]
	switch field {
	case "status":
		cluster.Status = value.(string)
	case "message":
		cluster.Message = value.(string)
	case "outputs":
		cluster.Outputs = value.([]byte)
	case "terraform_config":
		cluster.TerraformConfig = value.([]byte)
	case "terraform_state":
		cluster.TerraformState = value.([]byte)
	}
	dao.clustersMap[id] = cluster
	return nil
}

func (dao *ValidClusterDao) GetCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error) {
	return dao.clustersMap[id], nil
}

func (dao *ValidClusterDao) GetClusters(db *sqlx.DB, requestId string) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	for _, cluster := range dao.clustersMap {
		clusters = append(clusters, *cluster)
	}
	return clusters, nil
}

func (dao *ValidClusterDao) DeleteCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error) {
	if _, ok := dao.clustersMap[id]; !ok {
		return nil, errors.New("foo")
	} else {
		dao.clustersMap[id].Status = "destroying"
		return dao.clustersMap[id], nil
	}
}

type EmptyClusterDao struct {
	clustersMap map[string]*models.Cluster
}

func NewEmptyClusterDao() *EmptyClusterDao {
	return &EmptyClusterDao{}
}

func (dao *EmptyClusterDao) CreateCluster(db *sqlx.DB, config []byte, requestId string) (*models.Cluster, error) {
	return nil, errors.New("foo")
}

func (dao *EmptyClusterDao) UpdateClusterField(db *sqlx.DB, id string, field string, value interface{}, requestId string) error {
	return nil
}

func (dao *EmptyClusterDao) GetCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error) {
	return nil, errors.New("foo")
}

func (dao *EmptyClusterDao) GetClusters(db *sqlx.DB, requestId string) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	return clusters, nil
}

func (dao *EmptyClusterDao) DeleteCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error) {
	return nil, errors.New("foo")
}
