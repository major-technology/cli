package file

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd is the hidden `major file` group: host markdown/HTML files at a Major link.
var Cmd = &cobra.Command{
	Use:    "file",
	Short:  "Host markdown and HTML files at a Major link",
	Hidden: true,
	Args:   utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(pushCmd)
	Cmd.AddCommand(pullCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(renameCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(shareCmd)
}
