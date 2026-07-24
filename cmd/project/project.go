package project

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the project command group
var Cmd = &cobra.Command{
	Use:   "project",
	Short: "File-based project commands",
	Long:  `Commands for creating, validating, compiling, and deploying file-based major projects.`,
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

func init() {
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newValidateCmd())
	Cmd.AddCommand(newCompileCmd())
	Cmd.AddCommand(newViewCmd())
	Cmd.AddCommand(newDeployCmd())
}
