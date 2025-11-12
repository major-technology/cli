package git

import (
	"github.com/spf13/cobra"
)

// GitCmd represents the git command
var GitCmd = &cobra.Command{
	Use:   "git",
	Short: "Manage git configuration",
	Long:  `Manage git-related configuration for the Major CLI.`,
}

func init() {
	GitCmd.AddCommand(configCmd)
}

