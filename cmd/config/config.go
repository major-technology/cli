package config

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:    "config",
	Short:  "CLI configuration",
	Hidden: true,
	Args:   utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(setEnvCmd)
	Cmd.AddCommand(getEnvCmd)
}
