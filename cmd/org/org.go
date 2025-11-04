package org

import (
	"github.com/spf13/cobra"
)

// Cmd represents the org command
var Cmd = &cobra.Command{
	Use:   "org",
	Short: "Organization management commands",
	Long:  `Commands for managing organization selection and information.`,
}

func init() {
	// Add org subcommands
	Cmd.AddCommand(selectCmd)
	Cmd.AddCommand(whoamiCmd)
	Cmd.AddCommand(listCmd)
}
