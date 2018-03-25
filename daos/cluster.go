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
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "create_cluster",
		"request": requestId,
	})

	logger.Info("create cluster entry in database")

	if len(config) == 0 {
		logger.Error("cannot create cluster without config")
		logger.Error("failed to create cluster")
		return nil, errors.New("cannot create cluster without config")
	}

	cluster := models.Cluster{
		Name:            sillyname.GenerateStupidName(),
		Status:          models.ClusterStatusRequested,
		TerraformConfig: config,
	}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	logger.Debug("transaction created")

	logger.Debug(fmt.Sprintf(`
		INSERT INTO clusters 
		(name,status,terraform_config) 
		VALUES (%s,%s,%s) 
		RETURNING id`, cluster.Name, cluster.Status, cluster.TerraformConfig))
	rows, err := tx.NamedQuery(`
		INSERT INTO clusters 
		(name,status,terraform_config) 
		VALUES (:name,:status,:terraform_config) 
		RETURNING id`, cluster)
	if err != nil {
		tx.Rollback()
		logger.Debug("transaction rolledback")
		logger.Error(err.Error())
		logger.Error("failed to create cluster")
		return nil, err
	}
	defer rows.Close()

	var id string
	if rows.Next() {
		rows.Scan(&id)
	}
	cluster.Id = id

	tx.Commit()
	logger.Debug("transaction commited")
	logger.Debug(cluster)

	return &cluster, nil
}

func (dao *ClusterDao) UpdateCluster(db *sqlx.DB, cluster *models.Cluster, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "update_cluster",
		"request": requestId,
	})

	logger.Info("update cluster entry in database")

	updated_cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	logger.Debug("transaction created")

	logger.Debug(fmt.Sprintf("UPDATE clusters SET status = '%s' WHERE id = '%s' RETURNING *", cluster.Status, cluster.Id))
	rows, err := tx.Queryx(`UPDATE clusters SET status = $1 WHERE id = $2 RETURNING *`, &cluster.Status, &cluster.Id)
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
	logger.Debug(updated_cluster)

	logger.Info(fmt.Sprintf("cluster '%s' status '%s'", updated_cluster.Id, updated_cluster.Status))

	return &updated_cluster, nil
}

func (dao *ClusterDao) GetCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "get_cluster",
		"request": requestId,
	})

	logger.Info("get cluster from database")

	cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	logger.Debug("transaction created")

	logger.Debug(fmt.Sprintf("SELECT * FROM clusters WHERE id=%s", id))
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
		logger.Error(err.Error())
		return nil, err
	default:
		logger.Error(err.Error())
		logger.Error("could not get cluster")
		return nil, err
	}
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
	logger.Debug(updated_cluster)

	return &updated_cluster, nil
}
