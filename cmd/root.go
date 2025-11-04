package cmd

import (
	"os"

	apiClient "github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/config"
	"github.com/major-technology/cli/cmd/app"
	"github.com/major-technology/cli/cmd/org"
	"github.com/major-technology/cli/cmd/user"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	cfgFile       string
	version       = "dev"                // set by -ldflags
	defaultConfig = "configs/local.json" // can also be set by -ldflags
)

var rootCmd = &cobra.Command{
	Use:     "major",
	Short:   "The major CLI",
	Long:    `The major CLI is a tool to help you create and manage major applications`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Disable the completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// default comes from variable (override-able via ldflags)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfig, "config file")

	// Register subcommands
	rootCmd.AddCommand(user.Cmd)
	rootCmd.AddCommand(org.Cmd)
	rootCmd.AddCommand(app.Cmd)
}

func initConfig() {
	var err error
	cfg, err := config.Load(cfgFile, defaultConfig)
	cobra.CheckErr(err)

	// Set config in singletons package
	singletons.SetConfig(cfg)

	// Initialize API client with base URL (token will be fetched automatically per-request)
	client := apiClient.NewClient(cfg.APIURL)
	singletons.SetAPIClient(client)
}
