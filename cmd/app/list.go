package app

import (
	"fmt"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
	"text/tabwriter"
)

var listCmd = &cobra.Command{Use: "list", Short: "List accessible applications", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
	editable, _ := c.Flags().GetBool("editable")
	resp, err := singletons.GetAPIClient().ListApps(editable)
	if err != nil {
		return err
	}
	j, _ := c.Flags().GetBool("json")
	if j {
		return utils.WriteJSON(c, resp)
	}
	w := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tEditable")
	if apps, ok := resp["applications"].([]any); ok {
		for _, item := range apps {
			a, _ := item.(map[string]any)
			fmt.Fprintf(w, "%v\t%v\t%v\n", a["id"], a["name"], a["canEdit"])
		}
	}
	return w.Flush()
}}

func init() {
	listCmd.Flags().Bool("editable", false, "Only editable applications")
	listCmd.Flags().Bool("json", false, "Output route fields as JSON")
}
