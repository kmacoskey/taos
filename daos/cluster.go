package daos

import (
	"errors"
	"fmt"
	"regexp"

	sillyname "github.com/Pallinder/sillyname-go"
	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/models"
	log "github.com/sirupsen/logrus"
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

func (dao *ClusterDao) CreateCluster(tx *sqlx.Tx) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "create_cluster",
	})

	cluster := models.Cluster{
		Name:   sillyname.GenerateStupidName(),
		Status: "provisioning",
	}

	var id string
	rows, err := tx.NamedQuery(`INSERT INTO clusters (name,status) VALUES (:name,:status) RETURNING id`, cluster)
	if err == nil {
		if rows.Next() {
			rows.Scan(&id)
		}
		cluster.Id = id
		return &cluster, nil
	} else {
		logger.Debug(fmt.Sprintf("could not create cluster '%s'", err.Error()))
		return nil, err
	}

}

func (dao *ClusterDao) UpdateCluster(tx *sqlx.Tx, cluster *models.Cluster) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "update_cluster",
	})

	res, err := tx.Exec(`UPDATE clusters SET name = $2, status = $3 WHERE id = $1`, &cluster.Id, &cluster.Name, &cluster.Status)
	if err != nil {
		logger.Debug(fmt.Sprintf("could not update cluster '%s'", err.Error()))
		return nil, err
	}

	count, err := res.RowsAffected()
	if err != nil {
		logger.Debug(fmt.Sprintf("could not update cluster '%s'", err.Error()))
		return nil, err
	}

	if count != 1 {
		logger.Debug("no clusters updated")
		return nil, errors.New("no clusters updated")
	}

	return cluster, nil
}

func (dao *ClusterDao) GetCluster(tx *sqlx.Tx, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "GetCluster",
	})

	cluster := models.Cluster{}

	err := tx.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", id)
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

func (dao *ClusterDao) GetClusters(tx *sqlx.Tx) ([]models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "GetClusters",
	})

	clusters := []models.Cluster{}
	cluster := models.Cluster{}

	rows, err := tx.Queryx("SELECT * FROM clusters")

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
