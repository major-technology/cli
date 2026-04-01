package app

import (
	"encoding/json"
	"os"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:    "list",
	Short:  "List all applications in the current organization",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList()
	},
}

type appListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func runList() error {
	orgID, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()

	resp, err := apiClient.GetOrganizationApplications(orgID)
	if err != nil {
		return err
	}

	items := make([]appListItem, len(resp.Applications))
	for i, app := range resp.Applications {
		items[i] = appListItem{
			ID:   app.ID,
			Name: app.Name,
		}
	}

	return json.NewEncoder(os.Stdout).Encode(items)
}
