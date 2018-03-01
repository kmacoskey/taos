package models

type Cluster struct {
	Id              string `json:"id" db:"id"`
	Name            string `json:"name" db:"name"`
	Status          string `json:"status" db:"status"`
	TerraformConfig []byte `json:"terraform_config" db:"terraform_config"`
}
