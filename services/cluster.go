package services

import (
	"fmt"

	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/terraform"
)

type clusterDao interface {
	GetCluster(rc app.RequestContext, id string) (*models.Cluster, error)
	GetClusters(rc app.RequestContext) ([]models.Cluster, error)
	CreateCluster(rc app.RequestContext) (*models.Cluster, error)
}

type ClusterService struct {
	dao clusterDao
}

func NewClusterService(dao clusterDao) *ClusterService {
	return &ClusterService{dao}
}

func (s *ClusterService) GetCluster(rc app.RequestContext, id string) (*models.Cluster, error) {
	cluster, err := s.dao.GetCluster(rc, id)
	return cluster, err
}

func (s *ClusterService) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	clusters, err := s.dao.GetClusters(rc)
	return clusters, err
}

func (s *ClusterService) ProvisionCluster(*models.Cluster) {
	vtc := []byte(`{"provider":{"google":{}}}`)
	t := &terraform.Terraform{
		Config: vtc,
	}

	tc := terraform.Client{
		Terraform: t,
	}

	err := tc.ClientInit()
	if err != nil {
		fmt.Println(err)
	}

	err = tc.Apply()
	if err != nil {
		fmt.Println(err)
	}

	err = tc.ClientDestroy()
	if err != nil {
		fmt.Println(err)
	}

}

/*
 * func (s *ClusterService) SetClusterStatus(id, status string) error {
 *
 * }
 */

func (s *ClusterService) CreateCluster(rc app.RequestContext) (*models.Cluster, error) {
	cluster, err := s.dao.CreateCluster(rc)
	if err != nil {
		return cluster, err
	}

	go s.ProvisionCluster(cluster)

	cluster.Status = "provisioning"

	return cluster, err
}
