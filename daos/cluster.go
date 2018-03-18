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

func (dao *ClusterDao) CreateCluster(db *sqlx.DB, config []byte) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "create_cluster",
		"context": "nil",
	})

	logger.Debug("creating cluster in database")

	if len(config) == 0 {
		logger.Debug("Refusing to create cluster without config")
		return nil, errors.New("Refusing to create cluster without config")
	}

	cluster := models.Cluster{
		Name:            sillyname.GenerateStupidName(),
		Status:          "requested",
		TerraformConfig: config,
	}

	tx, err := db.Beginx()
	if err != nil {
		logger.Debug("Could not create transaction")
		return nil, errors.New("Could not create transaction")
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
		logger.Debug(fmt.Sprintf("could not create cluster '%s'", err.Error()))
		return nil, err
	}

	var id string
	if rows.Next() {
		rows.Scan(&id)
	}
	cluster.Id = id

	tx.Commit()
	logger.Debug("transaction commited")

	return &cluster, nil

}

func (dao *ClusterDao) UpdateCluster(db *sqlx.DB, cluster *models.Cluster) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "update_cluster",
		"context": cluster.Id,
	})

	logger.Debug("updating cluster in database")

	updated_cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
	}
	logger.Debug("transaction created")

	logger.Debug(fmt.Sprintf("UPDATE clusters SET status = '%s' WHERE id = '%s' RETURNING *", cluster.Status, cluster.Id))
	rows, err := tx.Queryx(`UPDATE clusters SET status = $1 WHERE id = $2 RETURNING *`, &cluster.Status, &cluster.Id)
	if err != nil {
		logger.Debug(fmt.Sprintf("could not update cluster status: '%v'", err.Error()))
		tx.Rollback()
		logger.Debug("transaction rolledback")
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.StructScan(&updated_cluster)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		logger.Debug("no clusters updated with query")
		tx.Rollback()
		logger.Debug("transaction rolledback")
		return nil, errors.New("no clusters updated")
	}

	tx.Commit()
	logger.Debug("transaction commited")

	return &updated_cluster, nil
}

func (dao *ClusterDao) GetCluster(db *sqlx.DB, id string) (*models.Cluster, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "daos",
		"event":   "get_cluster",
		"context": id,
	})

	logger.Debug("getting cluster from database")

	cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
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
		"data":    id,
	})

	logger.Debug("deleting cluster")

	updated_cluster := models.Cluster{}

	tx, err := db.Beginx()
	if err != nil {
		logger.Panic("Could not create transaction")
	}
	logger.Debug("transaction created")

	logger.Debug(fmt.Sprintf("UPDATE clusters SET status = '%s' WHERE id = '%s' AND status NOT IN ('destroyed') RETURNING *", models.ClusterStatusDestroying, id))
	rows, err := tx.Queryx(`UPDATE clusters SET status = $1 WHERE id = $2 AND status NOT IN ('destroyed') RETURNING *`, models.ClusterStatusDestroying, id)
	if err != nil {
		logger.Debug(fmt.Sprintf("could not update cluster status to 'destroyed': '%v'", err.Error()))
		tx.Rollback()
		logger.Debug("transaction rolledback")
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.StructScan(&updated_cluster)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		logger.Debug("no clusters destroyed")
		tx.Rollback()
		logger.Debug("transaction rolledback")
		return nil, errors.New("no clusters destroyed")
	}

	tx.Commit()
	logger.Debug("transaction commited")

	return &updated_cluster, nil
}
