package demo

import (
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the demo command
var Cmd = &cobra.Command{
	Use:   "demo",
	Short: "Demo application commands",
	Long:  `Commands for creating demo applications.`,
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
	// Add demo subcommands
	Cmd.AddCommand(createCmd)
}
