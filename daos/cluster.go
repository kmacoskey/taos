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

func (dao *ClusterDao) CreateCluster(db *sqlx.DB) (*models.Cluster, error) {
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

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
	}
	logger.Debug("transaction created")

	rows, err := tx.NamedQuery(`INSERT INTO clusters (name,status) VALUES (:name,:status) RETURNING id`, cluster)
	if err == nil {
		if rows.Next() {
			rows.Scan(&id)
		}
		cluster.Id = id

		tx.Commit()
		logger.Debug("transaction commited")

		return &cluster, nil
	} else {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug(fmt.Sprintf("could not create cluster '%s'", err.Error()))
		return nil, err
	}

}

func (dao *ClusterDao) UpdateCluster(db *sqlx.DB, cluster *models.Cluster) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "update_cluster",
	})

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
	}
	logger.Debug("transaction created")

	res, err := tx.Exec(`UPDATE clusters SET name = $2, status = $3 WHERE id = $1`, &cluster.Id, &cluster.Name, &cluster.Status)
	if err != nil {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug(fmt.Sprintf("could not update cluster '%s'", err.Error()))
		return nil, err
	}

	count, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug(fmt.Sprintf("could not update cluster '%s'", err.Error()))
		return nil, err
	}

	if count != 1 {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug("no clusters updated")
		return nil, errors.New("no clusters updated")
	}

	tx.Commit()
	logger.Debug("transaction commited")

	return cluster, nil
}

func (dao *ClusterDao) GetCluster(db *sqlx.DB, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "GetCluster",
	})

	cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
	}
	logger.Debug("transaction created")

	err = tx.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", id)
	if err == nil {
		tx.Commit()
		logger.Debug("transaction commited")
		return &cluster, nil
	}

	tx.Rollback()
	logger.Debug("transaction rolledback")

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

func (dao *ClusterDao) GetClusters(db *sqlx.DB) ([]models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "GetClusters",
	})

	clusters := []models.Cluster{}
	cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
	}
	logger.Debug("transaction created")

	rows, err := tx.Queryx("SELECT * FROM clusters")

	if err == nil {
		for rows.Next() {
			err := rows.StructScan(&cluster)
			if err != nil {
				logger.Error(err)
			}
			clusters = append(clusters, cluster)
		}
		tx.Commit()
		logger.Debug("transaction commited")

		return clusters, nil
	} else {
		tx.Rollback()
		logger.Debug("transaction rolledback")
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

func (dao *ClusterDao) DeleteCluster(db *sqlx.DB, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"context": "query",
		"event":   "delete_cluster",
	})

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
	}
	logger.Debug("transaction created")

	// Update cluster status to 'deleting' unless it is already 'deleted' or 'deleting'
	res, err := tx.Exec(`UPDATE clusters SET status = $2 WHERE id = $1 AND status NOT IN ('deleted', 'deleting')`, id, "deleting")
	if err != nil {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug(fmt.Sprintf("could not update cluster '%s' status to deleted", err.Error()))
		return nil, err
	}

	count, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug(fmt.Sprintf("could not update cluster '%s' status to deleted", err.Error()))
		return nil, err
	}

	if count != 1 {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug("no clusters updated")
		return nil, errors.New("no clusters updated")
	}

	cluster := models.Cluster{}

	err = tx.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", id)
	if err == nil {
		tx.Commit()
		logger.Debug("transaction commited")
		return &cluster, nil
	}

	tx.Rollback()
	logger.Debug("transaction rolledback")

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
