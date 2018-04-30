package daos

import (
	"errors"
	"fmt"
	"time"

	sillyname "github.com/Pallinder/sillyname-go"
	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/models"
	log "github.com/sirupsen/logrus"
)

type ClusterDao struct{}

func NewClusterDao() *ClusterDao {
	return &ClusterDao{}
}

func (dao *ClusterDao) CreateCluster(db *sqlx.DB, config []byte, timeout string, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "daos", "event": "create_cluster", "request": requestId})

	if len(config) == 0 {
		err := errors.New("cannot create cluster without config")
		logger.Error(err)
		return nil, err
	}

	if len(timeout) == 0 {
		err := errors.New("cannot create cluster without timeout")
		logger.Error(err)
		return nil, err
	}

	cluster := models.Cluster{
		Id:              requestId,
		Name:            sillyname.GenerateStupidName(),
		Status:          models.ClusterStatusRequested,
		Message:         "",
		TerraformConfig: config,
		Timestamp:       time.Now(),
		Timeout:         timeout,
	}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	sql := `INSERT INTO clusters (
		id,
		name,
		status,
		message,
		terraform_config,
		timestamp,
		timeout
	) VALUES (
			:id,
			:name,
			:status,
			:message,
			:terraform_config,
			:timestamp,
			:timeout
		)`
	_, err = tx.NamedQuery(sql, cluster)
	if err != nil {
		tx.Rollback()
		logger.Error(err.Error())
		return nil, err
	}

	tx.Commit()

	return &cluster, nil
}

func (dao *ClusterDao) GetCluster(db *sqlx.DB, id string, requestId string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "daos", "event": "get_cluster", "request": requestId})

	cluster := models.Cluster{}

	if len(requestId) == 0 {
		err := errors.New("cannot get cluster without requestId")
		logger.Error(err)
		return nil, err
	}

	if len(id) == 0 {
		err := errors.New("cannot get cluster without id")
		logger.Error(err)
		return nil, err
	}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	sql := `SELECT * FROM clusters WHERE id=$1`
	err = tx.Get(&cluster, sql, id)
	if err != nil {
		tx.Rollback()
		logger.Error(err.Error())
		return nil, err
	}

	tx.Commit()

	return &cluster, nil
}

func (dao *ClusterDao) GetClusters(db *sqlx.DB, requestId string) ([]models.Cluster, error) {
	logger := log.WithFields(log.Fields{"package": "daos", "event": "get_clusters", "request": requestId})

	if len(requestId) == 0 {
		err := errors.New("cannot get cluster without requestId")
		logger.Error(err)
		return nil, err
	}

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	sql := `SELECT * FROM clusters`
	rows, err := tx.Queryx(sql)
	if err != nil {
		tx.Rollback()
		logger.Error(err.Error())
		return nil, err
	}
	defer rows.Close()

	clusters := []models.Cluster{}
	cluster := models.Cluster{}

	for rows.Next() {
		err := rows.StructScan(&cluster)
		if err != nil {
			logger.Error(err)
			return nil, err
		}
		clusters = append(clusters, cluster)
	}

	tx.Commit()

	return clusters, nil
}

func (dao *ClusterDao) UpdateClusterField(db *sqlx.DB, id string, field string, value interface{}, requestId string) error {
	logger := log.WithFields(log.Fields{"package": "daos", "event": "update_cluster_status", "request": requestId})

	tx, err := db.Beginx()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	sql := ``
	switch field {
	case "status":
		sql = `UPDATE clusters SET status = $2 WHERE id = $1 `
	case "message":
		sql = `UPDATE clusters SET message = $2 WHERE id = $1 `
	case "outputs":
		sql = `UPDATE clusters SET outputs = $2 WHERE id = $1 `
	case "terraform_config":
		sql = `UPDATE clusters SET terraform_config = $2 WHERE id = $1 `
	case "terraform_state":
		sql = `UPDATE clusters SET terraform_state = $2 WHERE id = $1 `
	case "timeout":
		sql = `UPDATE clusters SET timeout = $2 WHERE id = $1 `
	case "timestamp":
		tx.Rollback()
		return errors.New("cannot update timestamp field")
	default:
		tx.Rollback()
		return errors.New(fmt.Sprintf("field '%s' does not exist", field))
	}

	result, err := tx.Exec(sql, id, value)
	if err != nil {
		tx.Rollback()
		logger.Error(err.Error())
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		logger.Error(err.Error())
		return err
	}

	if rows == 0 {
		tx.Rollback()
		return errors.New("no clusters updated")
	}

	tx.Commit()

	return nil
}
