package reaper

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/models"
	"github.com/kmacoskey/taos/services"
	"github.com/kmacoskey/taos/terraform"
	log "github.com/sirupsen/logrus"
)

type ClusterReaper struct {
	interval string
	ticker   *time.Ticker
	service  clusterService
	db       *sqlx.DB
}

type clusterService interface {
	DeleteCluster(request_id string, client services.TerraformClient, id string) (*models.Cluster, error)
	GetExpiredClusters(requestId string) ([]models.Cluster, error)
}

func NewClusterReaper(interval string, cluster_service clusterService, db *sqlx.DB) (*ClusterReaper, error) {
	logger := log.WithFields(log.Fields{"package": "app", "event": "new_reaper", "request": nil})

	duration, err := time.ParseDuration(interval)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	reaper := &ClusterReaper{
		interval: interval,
		service:  cluster_service,
		db:       db,
		ticker:   time.NewTicker(duration),
	}

	return reaper, nil
}

func (reaper *ClusterReaper) StartReaping() {
	logger := log.WithFields(log.Fields{"package": "app", "event": "reaping", "request": nil})

	go func() {
		for _ = range reaper.ticker.C {
			logger.Debug("reaping expired clusters")
			err := reaper.ReapClusters()
			if err != nil {
				logger.Error(err)
			}
		}
	}()
}

func (reaper *ClusterReaper) ReapClusters() error {
	request_id := uuid.Must(uuid.NewRandom()).String()
	logger := log.WithFields(log.Fields{"package": "app", "event": "reap_clusters", "request": request_id})

	clusters, err := reaper.ExpiredClusters(request_id)
	if err != nil {
		logger.Error(err)
		return err
	}

	for _, cluster := range clusters {
		logger.Info(fmt.Sprintf("reaping %v cluster(s)", len(clusters)))
		err := reaper.ReapCluster(cluster.Id)
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	return nil
}

func (reaper *ClusterReaper) ExpiredClusters(request_id string) ([]models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "app", "event": "get_expired_clusters", "request": request_id})

	clusters, err := reaper.service.GetExpiredClusters(request_id)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return clusters, nil
}

func (reaper *ClusterReaper) ReapCluster(id string) error {
	logger := log.WithFields(log.Fields{"package": "app", "event": "reap_cluster", "request": nil})

	if len(id) == 0 {
		err := errors.New("cannot reap a cluster without specifying an id")
		logger.Error(err)
		return err
	}

	_, err := reaper.service.DeleteCluster(id, terraform.NewTerraformClient(), id)
	if err != nil {
		return err
	}

	return nil
}
