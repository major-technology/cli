package config

import (
	"strings"

	"github.com/major-technology/cli/configs"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	APIURL             string `mapstructure:"api_url"`
	FrontendURI        string `mapstructure:"frontend_uri"`
	AppURLSuffix       string `mapstructure:"app_url_suffix"`
	AppURLFEOnlySuffix string `mapstructure:"app_url_fe_only_suffix"`
}

// Load initializes and returns the application config
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set environment variable prefix and enable automatic env
	v.SetEnvPrefix("MAJOR")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var configData []byte
	if configFile == "configs/prod.json" {
		configData = configs.ProdConfig
	} else {
		configData = configs.LocalConfig
	}

	// Set config type and read from embedded config
	v.SetConfigType("json")
	if err := v.ReadConfig(strings.NewReader(string(configData))); err != nil {
		return nil, err
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
