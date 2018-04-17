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
	UpdateClusterField(db *sqlx.DB, id string, field string, value interface{}, requestId string) error
	DeleteCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error)
}

type terraformClient interface {
	ClientInit() error
	ClientDestroy() error
	Init() (string, error)
	Plan() (string, error)
	Apply() ([]byte, string, error)
	Destroy() ([]byte, string, error)
	Outputs() ([]byte, error)
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

func (s *ClusterService) CreateCluster(context app.RequestContext, client *terraform.Client) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "services", "event": "create_cluster", "request": context.RequestId()})

	logger.Info("creating new cluster")

	cluster, err := s.dao.CreateCluster(s.db, context.TerraformConfig(), context.RequestId())
	if err != nil {
		return cluster, err
	}

	// Requested cluster is returned and eventual cluster status
	//  is handled in the terraform service asynchronously
	go s.TerraformProvisionCluster(client, cluster, context.TerraformConfig(), context.RequestId())

	return cluster, err
}

func (s *ClusterService) DeleteCluster(context app.RequestContext, client *terraform.Client, id string) (*models.Cluster, error) {
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

	logger.Info(fmt.Sprintf("cluster '%s' loaded", id))

	switch cluster.Status {
	case models.ClusterStatusDestroying, models.ClusterStatusDestroyed:
		logger.Error(fmt.Sprintf("cannot destroy cluster that is already '%s' or '%s'", models.ClusterStatusDestroying, models.ClusterStatusDestroyed))
		logger.Error("failed to destroy cluster")
		return nil, errors.New(fmt.Sprintf("cannot destroy cluster that is already '%s' or '%s'", models.ClusterStatusDestroying, models.ClusterStatusDestroyed))
	}

	logger.Info(fmt.Sprintf("cluster '%s' destruction begin", id))

	cluster.Status = models.ClusterStatusDestroying
	err = s.dao.UpdateClusterField(s.db, cluster.Id, "status", models.ClusterStatusDestroying, context.RequestId())
	if err != nil {
		logger.Error(err.Error())
		logger.Error("failed to destroy cluster")
		return cluster, err
	}

	// After setting cluster status in the database, begin the destruction
	go s.TerraformDestroyCluster(client, cluster, context.RequestId())

	//  Cluster is returned and then eventual cluster status
	//  is handled in the go thread
	return cluster, nil
}

func (s *ClusterService) TerraformDestroyCluster(client *terraform.Client, c *models.Cluster, requestId string) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "services",
		"event":   "terraform_destroy",
		"request": requestId,
	})

	logger.Info(fmt.Sprintf("terraform destroy cluster '%s'", c.Id))

	client.Terraform.Config = c.TerraformConfig
	client.Terraform.State = c.TerraformState

	err := client.ClientInit()
	if err != nil {
		c.Status = models.ClusterStatusDestroyFailed
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", models.ClusterStatusDestroyFailed, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return
	}

	state, output, err := client.Destroy()
	if err != nil {
		c.Status = models.ClusterStatusDestroyFailed
		c.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", models.ClusterStatusDestroyFailed, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return
	}

	err = client.ClientDestroy()
	if err != nil {
		c.Status = models.ClusterStatusDestroyFailed
		c.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", models.ClusterStatusDestroyFailed, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return
	}

	logger.Info(fmt.Sprintf("terraform destroy cluster '%s' success", c.Id))
	logger.Debug(output)

	if err == nil {
		c.Status = "destroyed"
		c.Message = output
		c.TerraformState = state
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", "destroyed", requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("cluster destroying failed")
		}
		return
	}

}

func (s *ClusterService) TerraformProvisionCluster(client *terraform.Client, c *models.Cluster, config []byte, requestId string) {
	logger := log.WithFields(log.Fields{"package": "services", "event": "create_cluster", "request": requestId})

	client.Terraform.Config = config

	c.Status = "provisioning"
	err := s.dao.UpdateClusterField(s.db, c.Id, "status", "provisioning", requestId)
	if err != nil {
		logger.Error(err.Error())
		logger.Error("failed to update cluster during provisionining")
	}

	err = client.ClientInit()
	if err != nil {
		c.Status = "provision_failed"
		logger.Error(err.Error())
		logger.Error("cluster provisioning failed during clientinit")
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", "provision_failed", requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during provision failure")
		}
		return
	}

	state, stdout, err := client.Apply()
	if err != nil {
		c.Status = "provision_failed"
		c.Message = err.Error()
		logger.Error(err.Error())
		logger.Error("cluster provisioning failed during terraform apply")
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", c.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during provision failure")
		}
		err = s.dao.UpdateClusterField(s.db, c.Id, "message", c.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during provision failure")
		}
		return
	}

	// The state must be set in the client
	//  in order to retrieve outputs
	client.Terraform.State = state

	outputs, err := client.Outputs()
	if err != nil {
		c.Status = "provision_failed"
		c.Message = err.Error()
		logger.Error(err.Error())
		logger.Error("cluster provisioning failed during terraform output")
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", c.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during terraform output")
		}
		err = s.dao.UpdateClusterField(s.db, c.Id, "message", c.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during provision failure")
		}
		return
	}

	err = client.ClientDestroy()
	if err != nil {
		c.Status = "provision_failed"
		c.Message = err.Error()
		logger.Error(err.Error())
		logger.Error("cluster provisioning failed during client cleanup")
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", c.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during terraform output")
		}
		err = s.dao.UpdateClusterField(s.db, c.Id, "message", c.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during provision failure")
		}
		return
	}

	if err == nil {
		c.Status = "provision_success"
		c.Message = stdout
		c.Outputs = outputs
		c.TerraformState = state
		err := s.dao.UpdateClusterField(s.db, c.Id, "status", c.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during terraform output")
		}
		err = s.dao.UpdateClusterField(s.db, c.Id, "message", c.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("failed to update cluster during provision failure")
		}
		return
	}

}
