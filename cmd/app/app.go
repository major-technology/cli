package app

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the app command
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application management commands",
	Long:  `Commands for creating and managing applications.`,
	Args:  utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add app subcommands
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(infoCmd)
	Cmd.AddCommand(generateEnvCmd)
	Cmd.AddCommand(generateResourcesCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(deployCmd)
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(cloneCmd)
}
