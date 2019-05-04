package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Timeout   string   `mapstructure:"timeout"`
	Endpoints []string `mapstructure:"endpoints"`
}

func GetConfig(configPath string) (*Config, error) {
	var config Config

	// Set the file path and read the config.
	filename := filepath.Base(configPath)
	ext := filepath.Ext(configPath)
	path := filepath.Dir(configPath)
	viper.SetConfigType(strings.TrimPrefix(ext, "."))
	viper.SetConfigName(strings.TrimSuffix(filename, ext))
	viper.AddConfigPath(path)
	if err := viper.ReadInConfig(); err != nil {
		return &config, err
	}
	viper.Unmarshal(&config)

	if err := validateConfig(&config); err != nil {
		return &config, err
	}
	return &config, nil
}

func validateConfig(config *Config) error {
	// Validate there is at least 1 endpoint in the config file.
	if len(config.Endpoints) == 0 {
		return fmt.Errorf("no endpoints specified")
	}

	// Validate Timeout.
	if _, err := time.ParseDuration(config.timeout); err != nil {
		return fmt.Errorf("invalid or missing 'timeout': %s", config.Timeout)
	}

	return nil
}
