package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

func DatabaseConnect() (*sql.DB, error) {
	// Validate connection string
	db, err := sql.Open("postgres", config.ConnStr)
	if err != nil {
		return nil, err
	}

	// Ensure connection to database is possible
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"event": "startup",
		"topic": "taos",
	}).Info("connection to database confirmed")

	return db, err
}
