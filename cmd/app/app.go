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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "list" {
			if root := cmd.Parent().Parent(); root != nil && root.PersistentPreRunE != nil {
				return root.PersistentPreRunE(cmd, args)
			}
			return nil
		}
		return middleware.ChainParent(
			middleware.CheckNodeInstalled,
			middleware.CheckNodeVersion("22.12"),
			middleware.CheckPnpmInstalled,
		)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

func init() {
	// Add app subcommands
	Cmd.AddCommand(aiProxyCmd)
	Cmd.AddCommand(cloneCmd)
	Cmd.AddCommand(configureCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deployCmd)
	Cmd.AddCommand(deployStatusCmd)
	Cmd.AddCommand(errorsCmd)
	Cmd.AddCommand(infoCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(logsCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(themeCmd)
}
