package resource

import (
	"github.com/spf13/cobra"
)

// Cmd represents the resource command
var Cmd = &cobra.Command{
	Use:   "resource",
	Short: "Resource management commands",
	Long:  `Commands for creating and managing resources.`,
}

func init() {
	// Add resource subcommands
	Cmd.AddCommand(createCmd)
}
