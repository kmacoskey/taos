package app

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func InitLogger(config LoggingConfig) error {
	switch config.Format {
	case "text":
		log.SetFormatter(&log.TextFormatter{})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	switch config.Level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.ErrorLevel)
	}

	// Output to stdout instead of the default stderr
	// Can be any io.Writer
	log.SetOutput(os.Stdout)

	log.WithFields(log.Fields{
		"event": "startup",
		"topic": "taos",
	}).Info("logging begins")

	log.WithFields(log.Fields{
		"event": "startup",
		"topic": "taos",
	}).Info("server configuration loaded")

	return nil
}
