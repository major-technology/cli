package config

import (
	"os"
	"strings"

	"github.com/major-technology/cli/configs"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	APIURL      string `mapstructure:"api_url"`
	FrontendURI string `mapstructure:"frontend_uri"`
}

// Load initializes and returns the application config
func Load(configFile, defaultConfig string) (*Config, error) {
	v := viper.New()

	// Set environment variable prefix and enable automatic env
	v.SetEnvPrefix("MAJOR")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Load embedded config based on defaultConfig
	var configData []byte
	if defaultConfig == "configs/prod.json" {
		configData = configs.ProdConfig
	} else {
		configData = configs.LocalConfig
	}

	// Set config type and read from embedded config
	v.SetConfigType("json")
	if err := v.ReadConfig(strings.NewReader(string(configData))); err != nil {
		return nil, err
	}

	// Merge user config file if specified and different from default
	if configFile != "" && configFile != defaultConfig {
		if _, err := os.Stat(configFile); err == nil {
			v.SetConfigFile(configFile)
			// Don't fail on merge error, just use embedded defaults
			_ = v.MergeInConfig()
		}
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
