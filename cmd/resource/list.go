package resource

import (
	"encoding/json"
	"fmt"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available resources",
	Long:  `List all resources in the organization, showing which are attached to the current app.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runList(cobraCmd)
	},
}

func runList(cobraCmd *cobra.Command) error {
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	apiClient := singletons.GetAPIClient()

	// Get all org resources
	orgResources, err := apiClient.GetResources(appInfo.OrganizationID)
	if err != nil {
		return errors.WrapError("failed to get resources", err)
	}

	// Get app-attached resources
	appResources, err := apiClient.GetApplicationResources(appInfo.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to get application resources", err)
	}

	// Build set of attached resource IDs
	attached := make(map[string]bool)
	for _, r := range appResources.Resources {
		attached[r.ID] = true
	}

	type resourceJSON struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
		IsAttached  bool   `json:"isAttached"`
	}

	resources := make([]resourceJSON, len(orgResources.Resources))
	for i, r := range orgResources.Resources {
		resources[i] = resourceJSON{
			ID:          r.ID,
			Name:        r.Name,
			Type:        r.Type,
			Description: r.Description,
			IsAttached:  attached[r.ID],
		}
	}

	data, err := json.Marshal(resources)
	if err != nil {
		return errors.WrapError("failed to marshal JSON", err)
	}
	fmt.Fprintln(cobraCmd.OutOrStdout(), string(data))
	return nil
}
