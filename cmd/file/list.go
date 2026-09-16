package file

import (
	"fmt"
	"text/tabwriter"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagListJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List hosted files you can open in your default organization",
	Args:  utils.NoArgs,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd)
	},
}

func init() {
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format")
}

func runList(cmd *cobra.Command) error {
	orgID, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return err
	}

	resp, err := singletons.GetAPIClient().ListFiles(orgID)
	if err != nil {
		return err
	}

	if flagListJSON {
		return utils.WriteJSON(cmd, resp.Files)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tKIND\tVERSION\tUPDATED\tLINK")
	for _, f := range resp.Files {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", f.ID, f.Name, f.Kind, f.Version, f.UpdatedAt, f.Link)
	}
	return w.Flush()
}
