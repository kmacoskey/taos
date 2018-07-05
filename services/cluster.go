package services

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	log "github.com/sirupsen/logrus"
)

type clusterDao interface {
	GetCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error)
	GetClusters(db *sqlx.DB, requestId string) ([]models.Cluster, error)
	GetExpiredClusters(db *sqlx.DB, requestId string) ([]models.Cluster, error)
	CreateCluster(db *sqlx.DB, config []byte, timeout string, requestId string, project string, region string) (*models.Cluster, error)
	UpdateClusterField(db *sqlx.DB, id string, field string, value interface{}, requestId string) error
}

type TerraformClient interface {
	Config() []byte
	SetConfig([]byte)
	State() []byte
	SetState([]byte)
	Project() string
	SetProject(string)
	Region() string
	SetRegion(string)
	Credentials() string
	SetCredentials(string)
	ClientInit() error
	ClientDestroy() error
	Init() (string, error)
	Plan(bool) (string, error)
	Apply() ([]byte, string, error)
	Destroy() ([]byte, string, error)
	Outputs() (string, error)
}

type ClusterService struct {
	dao clusterDao
	db  *sqlx.DB
}

func NewClusterService(dao clusterDao, db *sqlx.DB) *ClusterService {
	return &ClusterService{dao, db}
}

func (s *ClusterService) GetCluster(request_id string, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "services", "event": "get_cluster", "request": request_id})
	logger.Info(fmt.Sprintf("servicing request to get cluster '%v'", id))
	cluster, err := s.dao.GetCluster(s.db, id, request_id)
	logger.Info(fmt.Sprintf("service returning cluster '%v'", id))
	return cluster, err
}

func (s *ClusterService) GetClusters(request_id string) ([]models.Cluster, error) {
	clusters, err := s.dao.GetClusters(s.db, request_id)
	return clusters, err
}

func (s *ClusterService) GetExpiredClusters(request_id string) ([]models.Cluster, error) {
	clusters, err := s.dao.GetExpiredClusters(s.db, request_id)
	return clusters, err
}

func (s *ClusterService) CreateCluster(terraform_config []byte, timeout string, project string, region string, request_id string, client TerraformClient) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "services", "event": "create_cluster", "request": request_id})
	logger.Info("servicing request to create cluster")
	cluster, err := s.dao.CreateCluster(s.db, terraform_config, timeout, request_id, project, region)
	if err != nil {
		return cluster, err
	}
	credentials := app.GlobalServerConfig.Credentials(project)
	if len(credentials) == 0 {
		logger.Error(models.CredentialsNotFound)
	}

	client.SetCredentials(credentials)
	client.SetProject(project)
	client.SetRegion(region)

	// Cluster with requested action is returned and eventual cluster status
	//  is handled in the terraform service asynchronously
	go s.TerraformProvisionCluster(client, cluster, terraform_config, request_id)

	logger.Info("service returning requested cluster")

	return cluster, err
}

func (s *ClusterService) DeleteCluster(request_id string, client TerraformClient, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "services", "event": "delete_cluster", "request": request_id})

	logger.Info("servicing request to delete cluster")

	// Retrieve the cluster to destroy
	cluster, err := s.dao.GetCluster(s.db, id, request_id)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	if cluster == nil {
		err := errors.New("cannot destroy cluster that does not exist")
		logger.Error(err)
		return nil, err
	}

	if cluster.Status == models.ClusterStatusDestroying || cluster.Status == models.ClusterStatusDestroyed {
		err := errors.New(fmt.Sprintf("cannot destroy cluster that is already '%s' or '%s'", models.ClusterStatusDestroying, models.ClusterStatusDestroyed))
		return nil, err
	}

	cluster.Status = models.ClusterStatusDestroying
	err = s.dao.UpdateClusterField(s.db, cluster.Id, "status", models.ClusterStatusDestroying, request_id)
	if err != nil {
		logger.Error(err.Error())
		return cluster, err
	}

	credentials := app.GlobalServerConfig.Credentials(cluster.Project)
	if len(credentials) == 0 {
		logger.Error(models.CredentialsNotFound)
	}

	client.SetCredentials(credentials)
	client.SetProject(cluster.Project)
	client.SetRegion(cluster.Region)

	// Cluster with requested action is returned and eventual cluster status
	//  is handled in the terraform service asynchronously
	go s.TerraformDestroyCluster(client, cluster, request_id)

	logger.Info("servicing returning cluster set to delete")

	return cluster, nil
}

func (s *ClusterService) TerraformDestroyCluster(client TerraformClient, cluster *models.Cluster, requestId string) {
	logger := log.WithFields(log.Fields{"package": "services", "event": "terraform_destroy", "request": requestId})

	client.SetConfig(cluster.TerraformConfig)
	client.SetState(cluster.TerraformState)

	err := client.ClientInit()
	if err != nil {
		cluster.Status = models.ClusterStatusDestroyFailed
		cluster.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", models.ClusterStatusDestroyFailed, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return
	}

	state, output, err := client.Destroy()
	if err != nil {
		cluster.Status = models.ClusterStatusDestroyFailed
		cluster.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", models.ClusterStatusDestroyFailed, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return
	}

	err = client.ClientDestroy()
	if err != nil {
		cluster.Status = models.ClusterStatusDestroyFailed
		cluster.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", models.ClusterStatusDestroyFailed, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return
	}

	if err == nil {
		cluster.Status = models.ClusterStatusDestroyed
		cluster.Message = output
		cluster.TerraformState = state
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", cluster.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return
	}

}

func (s *ClusterService) TerraformProvisionCluster(client TerraformClient, cluster *models.Cluster, config []byte, requestId string) *models.Cluster {
	logger := log.WithFields(log.Fields{"package": "services", "event": "terraform_provision", "request": requestId})

	client.SetConfig(config)

	cluster.Status = models.ClusterStatusProvisionStart
	err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", cluster.Status, requestId)
	if err != nil {
		logger.Error(models.ClusterUpdateFailed)
	}

	err = client.ClientInit()
	if err != nil {
		logger.Error(models.ClusterProvisioningFailed)
		cluster.Status = models.ClusterStatusProvisionFailed
		cluster.Message = err.Error()
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", cluster.Status, requestId)
		if err != nil {
			logger.Error(models.ClusterUpdateFailed)
		}
		err = s.dao.UpdateClusterField(s.db, cluster.Id, "message", cluster.Message, requestId)
		if err != nil {
			logger.Error(models.ClusterUpdateFailed)
		}
		return cluster
	}

	state, stdout, err := client.Apply()
	if err != nil {
		cluster.Status = models.ClusterStatusProvisionFailed
		cluster.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", cluster.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
		}

		// Attempt to rollback
		rollback_state, rollback_stdout, err := client.Destroy()
		if err != nil {
			cluster.Status = models.ClusterStatusProvisionFailedRollbackFailed
			cluster.Message = cluster.Message + "\n" + err.Error()
			logger.Error(err.Error())
		} else {
			cluster.Status = models.ClusterStatusProvisionFailedRollbackSuccess
			cluster.Message = cluster.Message + "\n" + rollback_stdout
			cluster.TerraformState = rollback_state
		}

		err = s.dao.UpdateClusterField(s.db, cluster.Id, "message", cluster.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
		}

		return cluster
	}

	// The state must be set in the client
	//  in order to retrieve outputs
	client.SetState(state)

	outputs, err := client.Outputs()
	if err != nil {
		cluster.Status = models.ClusterStatusProvisionFailed
		cluster.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", cluster.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		err = s.dao.UpdateClusterField(s.db, cluster.Id, "message", cluster.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return cluster
	}

	err = client.ClientDestroy()
	if err != nil {
		cluster.Status = models.ClusterStatusProvisionFailed
		cluster.Message = err.Error()
		logger.Error(err.Error())
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", cluster.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		err = s.dao.UpdateClusterField(s.db, cluster.Id, "message", cluster.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
		}

		return cluster
	}

	if err == nil {
		cluster.Status = models.ClusterStatusProvisionSuccess
		cluster.Message = stdout
		cluster.Outputs = []byte(outputs)
		cluster.TerraformState = state
		err := s.dao.UpdateClusterField(s.db, cluster.Id, "status", cluster.Status, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		err = s.dao.UpdateClusterField(s.db, cluster.Id, "message", cluster.Message, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		err = s.dao.UpdateClusterField(s.db, cluster.Id, "outputs", cluster.Outputs, requestId)
		if err != nil {
			logger.Error(err.Error())
		}
		return cluster
	}

	return cluster
}
