package app

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

func DatabaseConnect(connStr string) (*sqlx.DB, error) {
	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "app",
		"context": "database",
		"event":   "startup",
	})

	// This Pings the database trying to connect, panics on error
	// use sqlx.Open() for sql.Open() semantics
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, err
	}

	logger.Info("connection to database confirmed")

	return db, err
}
