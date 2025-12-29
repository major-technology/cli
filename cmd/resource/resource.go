package resource

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the resource command
var Cmd = &cobra.Command{
	Use:   "resource",
	Short: "Resource management commands",
	Long:  `Commands for creating and managing resources.`,
	Args:  utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add resource subcommands
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(manageCmd)
	Cmd.AddCommand(envCmd)
}
