package daos

import (
	"github.com/kmacoskey/taos/app"
	"github.com/kmacoskey/taos/models"
)

type ClusterDAO struct{}

func GetCluster(rc app.RequestContext, id int) (models.Cluster, error) {
	cluster := models.Cluster{}
	err := rc.Tx().Get(&cluster, "SELECT * FROM clusters WHERE id=$1", id)
	return cluster, err
}
