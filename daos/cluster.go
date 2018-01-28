package daos

import (
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	log "github.com/sirupsen/logrus"
)

type ClusterDao struct{}

func NewClusterDao() *ClusterDao {
	return &ClusterDao{}
}

func (dao *ClusterDao) GetCluster(rc app.RequestContext, id int) (models.Cluster, error) {
	cluster := models.Cluster{}
	err := rc.Tx().Get(&cluster, "SELECT * FROM clusters WHERE id=$1", id)
	return cluster, err
}

func (dao *ClusterDao) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	cluster := models.Cluster{}
	rows, err := rc.Tx().Queryx("SELECT * FROM clusters")
	for rows.Next() {
		err := rows.StructScan(&cluster)
		if err != nil {
			log.WithFields(log.Fields{
				"topic": "taos",
				"event": "cluster_daos",
			}).Error(err)
		}
		clusters = append(clusters, cluster)
	}
	return clusters, err
}
