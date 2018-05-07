package app

import (
	"fmt"

	"github.com/spf13/viper"
)

var GlobalServerConfig ServerConfig

type ServerConfig struct {
	// Required - Defaults to 8080 - Listen and Serve port
	ServerPort int `mapstructure:"server_port"`

	// Required -  No Default - Database connection string. Must be supported by lib pq.
	ConnStr string `mapstructure:"conn_str"`

	// Required - Defaults to 15m - Interval to reap expired clusters
	ReapInterval string `mapstructure:"reap_interval"`

	// Logrus Configuration
	Logging LoggingConfig
}

type LoggingConfig struct {
	// Optional - Defaults to Text - Logrus formater
	Format string `mapstructure:"log_format"`

	// Optional - Defaults to Text - Only log when greater then set level
	// Possible Level: Debug, Info, Warning, Error, Fatal and Panic
	Level string `mapstructure:"log_level"`
}

// Load the server configuration from ConfigPath/Name.Type or from the ENV with TAOS_[var]
func LoadServerConfig(config *ServerConfig, path string) error {
	v := viper.New()

	// Search for configuration file at <path>/config.yml
	v.AddConfigPath(path)
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Allow ENV vars of TAOS_[var]
	v.SetEnvPrefix("taos")
	v.AutomaticEnv()

	// Set Defaults
	v.SetDefault("server_port", 8080)
	v.SetDefault("reap_interval", "15m")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("Failed to read the configuration file: %s", err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return err
	}

	// TODO: Validate configuration values

	return nil
}
