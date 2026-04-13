package mcp

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:    "mcp",
	Short:  "MCP server utilities",
	Hidden: true,
	Args:   utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(checkReadonlyCmd)
	Cmd.AddCommand(checkReadonlyHookCmd)
	Cmd.AddCommand(getHeadersCmd)
}
