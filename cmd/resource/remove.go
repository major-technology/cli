package resource

import (
	"fmt"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagRemoveResourceID string

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a resource from the current application",
	Long:  `Remove a resource by ID from the current application. Use 'major resource list' to see attached resources.`,
	PreRunE: middleware.ChainParent(
		middleware.CheckLogin,
		middleware.CheckNodeInstalled,
		middleware.CheckNodeVersion("22.12"),
		middleware.CheckPnpmInstalled,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runRemove(cobraCmd)
	},
}

func init() {
	removeCmd.Flags().StringVar(&flagRemoveResourceID, "id", "", "Resource ID to remove")
	removeCmd.MarkFlagRequired("id")
}

func runRemove(cobraCmd *cobra.Command) error {
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

	// Read local resources.json (same as manage does)
	existingResources, err := utils.ReadLocalResources(".")
	if err != nil {
		return errors.WrapError("failed to read existing resources", err)
	}

	// Verify the resource is currently attached
	found := false
	selectedIDs := make([]string, 0, len(existingResources))
	for _, r := range existingResources {
		if r.ID == flagRemoveResourceID {
			found = true
			continue
		}
		selectedIDs = append(selectedIDs, r.ID)
	}

	if !found {
		return fmt.Errorf("resource with ID %q is not attached to this application", flagRemoveResourceID)
	}

	// Save to server
	_, err = apiClient.SaveApplicationResources(appInfo.OrganizationID, appInfo.ApplicationID, selectedIDs)
	if err != nil {
		return errors.WrapError("failed to save resources", err)
	}

	// Build resource list for AddResourcesToProject (needs ResourceItem details)
	selectedResources := utils.ResolveResourceItems(selectedIDs, orgResources.Resources)

	// Generate local client code (diffs against resources.json, will remove the target)
	if err := utils.AddResourcesToProject(cobraCmd, ".", selectedResources, appInfo.ApplicationID); err != nil {
		return errors.WrapError("failed to update project resources", err)
	}

	return nil
}
