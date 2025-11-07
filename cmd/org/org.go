package org

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the org command
var Cmd = &cobra.Command{
	Use:   "org",
	Short: "Organization management commands",
	Long:  `Commands for managing organization selection and information.`,
	Args:  utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add org subcommands
	Cmd.AddCommand(selectCmd)
	Cmd.AddCommand(whoamiCmd)
	Cmd.AddCommand(listCmd)
}
