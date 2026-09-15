package file

import (
	"fmt"

	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <file-id> <new-name>",
	Short: "Rename a hosted file",
	Args:  cobra.ExactArgs(2),
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := singletons.GetAPIClient().RenameFile(args[0], args[1])
		if err != nil {
			return clierrors.WrapError("failed to rename file", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Renamed to %s\n", resp.File.Name)
		return nil
	},
}
