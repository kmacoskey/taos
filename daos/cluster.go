package daos

import (
	"fmt"
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
	log "github.com/sirupsen/logrus"
	"regexp"
)

var (
	noRelation *regexp.Regexp
)

type ClusterDao struct{}

func init() {
	noRelation, _ = regexp.Compile(`pq: relation ".*" does not exist`)
}

func NewClusterDao() *ClusterDao {
	return &ClusterDao{}
}

func (dao *ClusterDao) GetCluster(rc app.RequestContext, id int) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "GetCluster",
	})

	cluster := models.Cluster{}

	err := rc.Tx().Get(&cluster, "SELECT * FROM clusters WHERE id=$1", id)
	if err == nil {
		return &cluster, nil
	}

	switch {
	case noRelation.MatchString(err.Error()):
		logger.Error(err)
		return nil, err
	default:
		logger.Debug(fmt.Sprintf("could not retrieve cluster '%v'", id))
		logger.Debug(err)
		return nil, err
	}
}

func (dao *ClusterDao) GetClusters(rc app.RequestContext) ([]models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "GetClusters",
	})

	clusters := []models.Cluster{}
	cluster := models.Cluster{}

	rows, err := rc.Tx().Queryx("SELECT * FROM clusters")

	if err == nil {
		for rows.Next() {
			err := rows.StructScan(&cluster)
			if err != nil {
				logger.Error(err)
			}
			clusters = append(clusters, cluster)
		}

		return clusters, nil
	} else {
		switch {
		case noRelation.MatchString(err.Error()):
			logger.Debug(err)
			return nil, err
		default:
			logger.Debug("could not retrieve clusters")
			logger.Debug(err)
			return nil, err
		}
	}

}
