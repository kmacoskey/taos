package services

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/terraform"
	log "github.com/sirupsen/logrus"
)

type clusterDao interface {
	GetCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error)
	GetClusters(db *sqlx.DB, requestId string) ([]models.Cluster, error)
	CreateCluster(db *sqlx.DB, config []byte, requestId string) (*models.Cluster, error)
	UpdateCluster(db *sqlx.DB, cluster *models.Cluster, requestId string) (*models.Cluster, error)
	DeleteCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error)
}

type ClusterService struct {
	dao clusterDao
	db  *sqlx.DB
}

func NewClusterService(dao clusterDao, db *sqlx.DB) *ClusterService {
	return &ClusterService{dao, db}
}

func (s *ClusterService) GetCluster(context app.RequestContext, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "services",
		"event":   "get_cluster",
		"request": context.RequestId(),
	})

	logger.Info("retrieving cluster from database")

	cluster, err := s.dao.GetCluster(s.db, id, context.RequestId())
	return cluster, err
}

func (s *ClusterService) GetClusters(context app.RequestContext) ([]models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "services",
		"event":   "get_clusters",
		"request": context.RequestId(),
	})

	logger.Info("retrieving clusters from database")

	clusters, err := s.dao.GetClusters(s.db, context.RequestId())
	return clusters, err
}

func (s *ClusterService) CreateCluster(context app.RequestContext) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "services",
		"event":   "create_cluster",
		"request": context.RequestId(),
	})

	logger.Info("creating new cluster")

	cluster, err := s.dao.CreateCluster(s.db, context.TerraformConfig(), context.RequestId())
	if err != nil {
		return cluster, err
	}

	// After creating new cluster entry in database, begin the provisioning
	go s.TerraformProvisionCluster(cluster, context.TerraformConfig(), context.RequestId())

	//  Requested cluster is returned and then eventual cluster status
	//  is handled in the go thread
	return cluster, err
}

func (s *ClusterService) DeleteCluster(context app.RequestContext, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "services",
		"event":   "delete_cluster",
		"request": context.RequestId(),
	})

	// Retrieve the cluster to destroy
	cluster, err := s.dao.GetCluster(s.db, id, context.RequestId())
	if err != nil {
		logger.Error(err.Error())
		logger.Error("failed to destroy cluster")
		return nil, err
	}

	if cluster == nil {
		logger.Error("cannot destroy cluster that does not exist")
		logger.Error("failed to destroy cluster")
		return nil, errors.New("cannot destroy cluster that does not exist")
	}

	switch cluster.Status {
	case models.ClusterStatusDestroying, models.ClusterStatusDestroyed:
		logger.Error(fmt.Sprintf("cannot destroy cluster that is already '%s' or '%s'", models.ClusterStatusDestroying, models.ClusterStatusDestroyed))
		logger.Error("failed to destroy cluster")
		return nil, errors.New(fmt.Sprintf("cannot destroy cluster that is already '%s' or '%s'", models.ClusterStatusDestroying, models.ClusterStatusDestroyed))
	}

	logger.Info("destroying cluster")

	cluster.Status = models.ClusterStatusDestroying
	cluster_being_destroyed, err := s.dao.DeleteCluster(s.db, id, context.RequestId())
	if err != nil {
		logger.Error(err.Error())
		logger.Error("failed to destroy cluster")
		return cluster_being_destroyed, err
	}

	// After setting cluster status in the database, begin the destruction
	go s.TerraformDestroyCluster(cluster_being_destroyed, context.RequestId())

	//  Cluster is returned and then eventual cluster status
	//  is handled in the go thread
	return cluster_being_destroyed, nil
}

func (s *ClusterService) TerraformDestroyCluster(c *models.Cluster, requestId string) {

	t := &terraform.Terraform{
		Config: c.TerraformConfig,
	}

	tc := terraform.Client{
		Terraform: t,
	}

	err := tc.ClientInit()
	if err != nil {
		c.Status = "destruction_failed at init"
		fmt.Println(err)
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	err = tc.Destroy()
	if err != nil {
		c.Status = "destruction_failed at destroy"
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	err = tc.ClientDestroy()
	if err != nil {
		c.Status = "destruction_failed at client destroy"
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	if err == nil {
		c.Status = "destroyed"
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

}

func (s *ClusterService) TerraformProvisionCluster(c *models.Cluster, config []byte, requestId string) {

	t := &terraform.Terraform{
		Config: config,
	}

	tc := terraform.Client{
		Terraform: t,
	}

	err := tc.ClientInit()
	if err != nil {
		c.Status = "provision_failed"
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	err = tc.Apply()
	if err != nil {
		c.Status = "provision_failed"
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	err = tc.ClientDestroy()
	if err != nil {
		c.Status = "provision_failed"
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	if err == nil {
		c.Status = "provision_success"
		_, err := s.dao.UpdateCluster(s.db, c, requestId)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

}
