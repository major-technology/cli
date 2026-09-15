package file

import (
	"fmt"
	"strings"

	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	flagShareEmail  string
	flagShareAccess string
)

var shareCmd = &cobra.Command{
	Use:   "share <file-id> --email <email> --access <viewer|editor|admin>",
	Short: "Grant a member of your organization access to a hosted file",
	Args:  cobra.ExactArgs(1),
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		role, err := roleForAccess(flagShareAccess)
		if err != nil {
			return err
		}
		if err := singletons.GetAPIClient().ShareFileByEmail(args[0], flagShareEmail, role); err != nil {
			return clierrors.WrapError("failed to share file", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Granted %s to %s\n", strings.ToLower(flagShareAccess), flagShareEmail)
		return nil
	},
}

func init() {
	shareCmd.Flags().StringVar(&flagShareEmail, "email", "", "Email of an existing organization member")
	shareCmd.Flags().StringVar(&flagShareAccess, "access", "viewer", "Access level: viewer, editor, or admin")
	_ = shareCmd.MarkFlagRequired("email")
}

// roleForAccess maps the user-facing access word to the API role name.
func roleForAccess(access string) (string, error) {
	switch strings.ToLower(access) {
	case "viewer":
		return "File:Viewer", nil
	case "editor":
		return "File:Editor", nil
	case "admin":
		return "File:Admin", nil
	default:
		return "", &clierrors.CLIError{
			Title:      fmt.Sprintf("unknown access level %q", access),
			Suggestion: "Use one of: viewer, editor, admin.",
		}
	}
}
