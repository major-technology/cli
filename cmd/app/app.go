package app

import (
	"github.com/spf13/cobra"
)

// Cmd represents the app command
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application management commands",
	Long:  `Commands for creating and managing applications.`,
}

func init() {
	// Add app subcommands
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(infoCmd)
	Cmd.AddCommand(generateEnvCmd)
	Cmd.AddCommand(generateResourcesCmd)
}
