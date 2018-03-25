package models

type Cluster struct {
	Id              string `json:"id" db:"id"`
	Name            string `json:"name" db:"name"`
	Status          string `json:"status" db:"status"`
	Message         string `json:"message" db:"message"`
	TerraformConfig []byte `json:"terraform_config" db:"terraform_config"`
	TerraformState  []byte `json:terraform_state" db:"terraform_state"`
}

const (
	ClusterStatusRequested  = "requested"
	ClusterStatusDestroying = "destroying"
	ClusterStatusDestroyed  = "destroyed"
)
