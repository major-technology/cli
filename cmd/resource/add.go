package resource

import (
	"fmt"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagAddResourceID string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a resource to the current application",
	Long:  `Add a resource by ID to the current application. Use 'major resource list' to see available resources.`,
	PreRunE: middleware.ChainParent(
		middleware.CheckLogin,
		middleware.CheckNodeInstalled,
		middleware.CheckNodeVersion("22.12"),
		middleware.CheckPnpmInstalled,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runAdd(cobraCmd)
	},
}

func init() {
	addCmd.Flags().StringVar(&flagAddResourceID, "id", "", "Resource ID to add")
	addCmd.MarkFlagRequired("id")
}

func runAdd(cobraCmd *cobra.Command) error {
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	apiClient := singletons.GetAPIClient()

	// Fetch org resources (source of truth for resource metadata)
	orgResources, err := apiClient.GetResources(appInfo.OrganizationID)
	if err != nil {
		return errors.WrapError("failed to get resources", err)
	}

	// Find the target resource in org resources
	var targetResource *api.ResourceItem
	for i, r := range orgResources.Resources {
		if r.ID == flagAddResourceID {
			targetResource = &orgResources.Resources[i]
			break
		}
	}

	if targetResource == nil {
		return fmt.Errorf("resource with ID %q not found in organization", flagAddResourceID)
	}

	// Read local resources.json (same as manage does)
	existingResources, err := utils.ReadLocalResources(".")
	if err != nil {
		cobraCmd.Printf("Warning: Could not read existing resources: %v\n", err)
		existingResources = []utils.LocalResource{}
	}

	// Build desired resource ID list: existing + new
	selectedIDs := make([]string, 0, len(existingResources)+1)
	for _, r := range existingResources {
		selectedIDs = append(selectedIDs, r.ID)
	}
	selectedIDs = append(selectedIDs, flagAddResourceID)

	// Save to server
	_, err = apiClient.SaveApplicationResources(appInfo.OrganizationID, appInfo.ApplicationID, selectedIDs)
	if err != nil {
		return errors.WrapError("failed to save resources", err)
	}

	// Build full resource list for AddResourcesToProject (needs ResourceItem details)
	selectedResources := utils.ResolveResourceItems(selectedIDs, orgResources.Resources)

	// Generate local client code (diffs against resources.json)
	if err := utils.AddResourcesToProject(cobraCmd, ".", selectedResources, appInfo.ApplicationID); err != nil {
		return errors.WrapError("failed to add resource to project", err)
	}

	return nil
}
