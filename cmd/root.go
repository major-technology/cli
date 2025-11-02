package cmd

import (
	"os"

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
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cli")
	}
	cobra.CheckErr(viper.ReadInConfig())
}
