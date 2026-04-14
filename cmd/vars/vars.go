package vars

import (
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the vars command
var Cmd = &cobra.Command{
	Use:   "vars",
	Short: "Manage environment variables",
	Long: `Manage environment variables for the current application.

Variables are scoped per-environment (e.g. development, staging, production).
Run these commands from inside a linked application directory.`,
	Args: utils.NoArgs,
	PersistentPreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(setCmd)
	Cmd.AddCommand(unsetCmd)
	Cmd.AddCommand(pullCmd)
}
