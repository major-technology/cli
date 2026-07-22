package project

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// CLIVersion is stamped by cmd/root.go from the ldflags-set version. It is
// recorded as compilerVersion on compile reports.
var CLIVersion = "dev"

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
}
