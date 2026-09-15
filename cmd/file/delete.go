package file

import (
	"fmt"

	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <file-id>",
	Short: "Delete a hosted file; its link stops working",
	Args:  cobra.ExactArgs(1),
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := singletons.GetAPIClient().DeleteFile(args[0]); err != nil {
			return clierrors.WrapError("failed to delete file", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Deleted")
		return nil
	},
}
