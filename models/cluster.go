package models

import (
	"time"
)

type Cluster struct {
	Id              string    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Status          string    `json:"status" db:"status"`
	Message         string    `json:"message" db:"message"`
	Outputs         []byte    `json:"outputs" db:"outputs"`
	TerraformConfig []byte    `json:"terraform_config" db:"terraform_config"`
	TerraformState  []byte    `json:"terraform_state" db:"terraform_state"`
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	Expiration      time.Time `json:"expiration" db:"expiration"`
	Timeout         string    `json:"timeout" db:"timeout"`
	Project         string    `json:"project" db:"project"`
	Region          string    `json:"region" db:"region"`
}

type Output struct {
	Sensitive string `json:"sensitive" db:"sensitive"`
	Type      string `json:"type" db:"type"`
	Value     string `json:"value" db:"value"`
}

const (
	ClusterStatusProvisionSuccess = "provision_success"
	ClusterStatusProvisionFailed  = "provision_failed"
	ClusterStatusProvisionStart   = "provisioning"
	ClusterStatusRequested        = "requested"
	ClusterStatusDestroying       = "destroying"
	ClusterStatusDestroyed        = "destroyed"
	ClusterStatusDestroyFailed    = "destruction_failed"
	ClusterUpdateFailed           = "failed to update cluster"
	ClusterProvisioningFailed     = "failed to provision cluster"
	CredentialsNotFound           = "credentials not found for the given project"
)
