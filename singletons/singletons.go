package singletons

import (
	apiClient "github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/config"
)

var (
	cfg    *config.Config
	client *apiClient.Client
)

// SetConfig sets the global configuration
func SetConfig(c *config.Config) {
	cfg = c
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return cfg
}

// SetAPIClient sets the global API client
func SetAPIClient(c *apiClient.Client) {
	client = c
}

// GetAPIClient returns the global API client
func GetAPIClient() *apiClient.Client {
	return client
}
