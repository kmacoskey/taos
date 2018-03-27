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

type ClusterDao struct{}

var (
	noRelation *regexp.Regexp
)

func init() {
	noRelation, _ = regexp.Compile(`pq: relation ".*" does not exist`)
}

func NewClusterDao() *ClusterDao {
	return &ClusterDao{}
}

func (dao *ClusterDao) CreateCluster(db *sqlx.DB, config []byte, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "daos", "event": "create_cluster", "request": requestId})

	if len(config) == 0 {
		logger.Error("cannot create cluster without config")
		return nil, errors.New("cannot create cluster without config")
	}

	cluster := models.Cluster{
		Id:              requestId,
		Name:            sillyname.GenerateStupidName(),
		Status:          models.ClusterStatusRequested,
		TerraformConfig: config,
	}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	_, err = tx.NamedQuery(`
		INSERT INTO clusters (id,name,status,terraform_config) 
		VALUES (:id,:name,:status,:terraform_config) `, cluster)
	if err != nil {
		tx.Rollback()
		logger.Error(err.Error())
		return nil, err
	}

	tx.Commit()
	logger.Info("new cluster created in database")

	return &cluster, nil
}

func (dao *ClusterDao) GetCluster(db *sqlx.DB, clusterId string, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "daos", "event": "get_cluster", "request": requestId})

	cluster := models.Cluster{}

	if len(requestId) == 0 {
		logger.Error("cannot get cluster without requestId")
		return nil, errors.New("cannot get cluster without requestId")
	} else if len(clusterId) == 0 {
		logger.Error("cannot get cluster without clusterId")
		return nil, errors.New("cannot get cluster without clusterId")
	}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	err = tx.Get(&cluster, "SELECT * FROM clusters WHERE id=$1", clusterId)
	if err != nil {
		tx.Rollback()
		logger.Error(err.Error())
		return nil, err
	}

	tx.Commit()

	logger.Info(fmt.Sprintf("cluster '%s' retrieved", clusterId))

	return &cluster, nil
}

func (dao *ClusterDao) UpdateCluster(db *sqlx.DB, cluster *models.Cluster, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "update_cluster",
		"request": requestId,
	})

	logger.Info("updating cluster")

	updated_cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	logger.Debug("transaction created")

	logger.Debug(fmt.Sprintf("UPDATE clusters SET status = '%s', message = '%s', terraform_state = '%s', outputs = '%s' WHERE id = '%s' RETURNING *", cluster.Status, cluster.Message, cluster.TerraformState, cluster.Outputs, cluster.Id))
	rows, err := tx.Queryx(`UPDATE clusters SET status = $1, message = $2, terraform_state = $3, outputs = $4 WHERE id = $5 RETURNING *`, &cluster.Status, &cluster.Message, &cluster.TerraformState, &cluster.Outputs, &cluster.Id)
	if err != nil {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Error(err.Error())
		logger.Error("failed to update cluster")
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.StructScan(&updated_cluster)
		if err != nil {
			tx.Rollback()
			logger.Debug("transaction rolledback")
			logger.Error(err.Error())
			logger.Error("failed to update cluster")
			return nil, err
		}
	} else {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Error("no clusters updated")
		return nil, errors.New("no clusters updated")
	}

	tx.Commit()
	logger.Debug("transaction commited")
	// logger.Debug(updated_cluster)

	logger.Info(fmt.Sprintf("cluster '%s' status '%s'", updated_cluster.Id, updated_cluster.Status))

	return &updated_cluster, nil
}

func (dao *ClusterDao) GetClusters(db *sqlx.DB, requestId string) ([]models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "get_clusters",
		"request": requestId,
	})

	clusters := []models.Cluster{}
	cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	logger.Debug("transaction created")

	logger.Debug("SELECT * FROM clusters")
	rows, err := tx.Queryx("SELECT * FROM clusters")
	if err == nil {
		for rows.Next() {
			err := rows.StructScan(&cluster)
			if err != nil {
				logger.Error(err)
			}
			clusters = append(clusters, cluster)
		}
	} else {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		switch {
		case noRelation.MatchString(err.Error()):
			logger.Error(err.Error())
			return nil, err
			logger.Error("could not retrieve clusters")
		default:
			logger.Debug(err.Error())
			logger.Error("could not retrieve clusters")
			return nil, err
		}
	}

	tx.Commit()
	logger.Debug("transaction commited")
	return clusters, nil
}

func (dao *ClusterDao) DeleteCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "delete_clusters",
		"request": requestId,
	})

	logger.Info("deleting cluster")

	updated_cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	logger.Debug("transaction created")

	logger.Debug(fmt.Sprintf("UPDATE clusters SET status = '%s' WHERE id = '%s' AND status NOT IN ('destroyed') RETURNING *", models.ClusterStatusDestroying, id))
	rows, err := tx.Queryx(`UPDATE clusters SET status = $1 WHERE id = $2 AND status NOT IN ('destroyed') RETURNING *`, models.ClusterStatusDestroying, id)
	if err != nil {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Error(err.Error())
		logger.Error(fmt.Sprintf("could not update cluster status to '%s'", models.ClusterStatusDestroying))
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.StructScan(&updated_cluster)
		if err != nil {
			tx.Rollback()
			logger.Error("transaction rolledback")
			logger.Error(err.Error())
			logger.Error("no clusters destroyed")
			return nil, err
		}
	} else {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Debug("no clusters destroyed")
		return nil, errors.New("no clusters destroyed")
	}

	tx.Commit()
	logger.Debug("transaction commited")
	// logger.Debug(updated_cluster)

	return &updated_cluster, nil
}
