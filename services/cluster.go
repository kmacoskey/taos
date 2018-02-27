package services

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/terraform"
)

type clusterDao interface {
	GetCluster(db *sqlx.DB, id string) (*models.Cluster, error)
	GetClusters(db *sqlx.DB) ([]models.Cluster, error)
	CreateCluster(db *sqlx.DB) (*models.Cluster, error)
	UpdateCluster(db *sqlx.DB, cluster *models.Cluster) (*models.Cluster, error)
	DeleteCluster(db *sqlx.DB, id string) (*models.Cluster, error)
}

type ClusterService struct {
	dao clusterDao
	db  *sqlx.DB
}

func NewClusterService(dao clusterDao, db *sqlx.DB) *ClusterService {
	return &ClusterService{dao, db}
}

func (s *ClusterService) GetCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	cluster, err := s.dao.GetCluster(s.db, id)
	return cluster, err
}

func (s *ClusterService) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	clusters, err := s.dao.GetClusters(s.db)
	return clusters, err
}

func (s *ClusterService) DeleteCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	return s.dao.DeleteCluster(s.db, id)
}

func (s *ClusterService) ProvisionCluster(c *models.Cluster, config []byte) {

	t := &terraform.Terraform{
		Config: config,
	}

	tc := terraform.Client{
		Terraform: t,
	}

	err := tc.ClientInit()
	if err != nil {
		c.Status = "provision_failed"
		_, err := s.dao.UpdateCluster(s.db, c)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	err = tc.Apply()
	if err != nil {
		c.Status = "provision_failed"
		_, err := s.dao.UpdateCluster(s.db, c)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	err = tc.ClientDestroy()
	if err != nil {
		c.Status = "provision_failed"
		_, err := s.dao.UpdateCluster(s.db, c)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	if err == nil {
		c.Status = "provision_success"
		_, err := s.dao.UpdateCluster(s.db, c)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

}

func (s *ClusterService) CreateCluster(rc app.RequestContext) (*models.Cluster, error) {
	cluster, err := s.dao.CreateCluster(s.db)
	if err != nil {
		return cluster, err
	}

	go s.ProvisionCluster(cluster, rc.TerraformConfig())

	cluster.Status = "provisioning"

	return cluster, err
}
