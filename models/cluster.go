package models

import ()

type Cluster struct {
	Id     int    `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	Status string `json:"status" db:"status"`
}
