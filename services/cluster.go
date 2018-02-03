package services

import (
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
)

type clusterDao interface {
	GetCluster(rc app.RequestContext, id int) (*models.Cluster, error)
	GetClusters(rc app.RequestContext) ([]models.Cluster, error)
}

type ClusterService struct {
	dao clusterDao
}

func NewClusterService(dao clusterDao) *ClusterService {
	return &ClusterService{dao}
}

func (s *ClusterService) GetCluster(rc app.RequestContext, id int) (*models.Cluster, error) {
	cluster, err := s.dao.GetCluster(rc, id)
	return cluster, err
}

func (s *ClusterService) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	clusters, err := s.dao.GetClusters(rc)
	return clusters, err
}
