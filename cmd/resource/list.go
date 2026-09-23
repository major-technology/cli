package resource

import (
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagListJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available resources",
	Long:  `List resources in the current organization.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runList(cobraCmd)
	},
}

func init() {
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format")
}

func runList(cobraCmd *cobra.Command) error {
	resp, err := singletons.GetAPIClient().ListResources()
	if err != nil {
		return err
	}

	return utils.WriteJSON(cobraCmd, resp.Resources)
}
