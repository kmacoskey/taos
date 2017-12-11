package main

import (
	"fmt"
	"github.com/spf13/viper"
)

type serverConfig struct {
	// Required - Defaults to 8080 - Listen and Serve port
	ServerPort int `mapstructure:"server_port"`

	// Required -  No Default - Database connection string. Must be supported by lib pq.
	ConnStr string `mapstructure:"conn_str"`

	// Optional - Defaults to Text - Logrus formater
	LogFormat string `mapstructure:"log_format"`

	// Optional - Defaults to Text - Only log when greater then set level
	// Possible Level: Debug, Info, Warning, Error, Fatal and Panic
	LogLevel string `mapstructure:"log_level"`
}

// Load the server configuration from ConfigPath/Name.Type or from the ENV with TAOS_[var]
func LoadServerConfig(config *serverConfig) error {
	v := viper.New()

	// Search for configuration file at ./config.yml
	v.AddConfigPath(".")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Allow ENV vars of TAOS_[var]
	v.SetEnvPrefix("taos")
	v.AutomaticEnv()

	// Set Defaults
	v.SetDefault("server_port", 8080)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("Failed to read the configuration file: %s", err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return err
	}

	// TODO: Validate configuration values

	return nil
}
