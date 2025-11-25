package app

import (
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the app command
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application management commands",
	Long:  `Commands for creating and managing applications.`,
	Args:  utils.NoArgs,
	PersistentPreRunE: middleware.ChainParent(
		middleware.CheckNodeInstalled,
		middleware.CheckNodeVersion("22.12"),
		middleware.CheckPnpmInstalled,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

func init() {
	// Add app subcommands
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(infoCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(deployCmd)
	Cmd.AddCommand(configureCmd)
	Cmd.AddCommand(cloneCmd)
}
