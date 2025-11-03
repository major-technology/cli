package cmd

import (
	"encoding/json"
	"os"

	"github.com/major-technology/cli/configs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	version       = "dev"                // set by -ldflags
	defaultConfig = "configs/local.json" // can also be set by -ldflags
)

var rootCmd = &cobra.Command{
	Use:     "cli",
	Short:   "The major CLI",
	Long:    `The major CLI is a tool to help you create and manage major applications`,
	Version: version, // you can set here OR in Execute(); here is simpler
}

func Execute() {
	// If you prefer, you can keep this:
	// rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// default comes from variable (override-able via ldflags)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfig, "config file")

	viper.SetEnvPrefix("MAJOR")
	viper.AutomaticEnv()
}

func initConfig() {
	// Use embedded config based on defaultConfig variable
	var configData []byte
	if defaultConfig == "configs/prod.json" {
		configData = configs.ProdConfig
	} else {
		configData = configs.LocalConfig
	}

	// Parse embedded config into viper
	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		cobra.CheckErr(err)
	}

	for key, value := range config {
		viper.Set(key, value)
	}

	// Allow user to override with custom config file if specified
	if cfgFile != "" && cfgFile != defaultConfig {
		if _, err := os.Stat(cfgFile); err == nil {
			viper.SetConfigFile(cfgFile)
			// Don't exit on error, just use embedded defaults
			_ = viper.MergeInConfig()
		}
	}

	viper.SetEnvPrefix("MAJOR")
	viper.AutomaticEnv()
}
