package resource

import (
	"fmt"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagRemoveResourceID string
	flagRemoveJSON       bool
)

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
	removeCmd.Flags().BoolVar(&flagRemoveJSON, "json", false, "Output in JSON format")
	removeCmd.MarkFlagRequired("id")
}

func runRemove(cobraCmd *cobra.Command) error {
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	apiClient := singletons.GetAPIClient()

	orgResources, err := apiClient.GetResources(appInfo.OrganizationID)
	if err != nil {
		return errors.WrapError("failed to get resources", err)
	}

	existingResources, err := utils.ReadLocalResources(".")
	if err != nil {
		return errors.WrapError("failed to read existing resources", err)
	}

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

	_, err = apiClient.SaveApplicationResources(appInfo.OrganizationID, appInfo.ApplicationID, selectedIDs)
	if err != nil {
		return errors.WrapError("failed to save resources", err)
	}

	selectedResources := utils.ResolveResourceItems(selectedIDs, orgResources.Resources)
	if err := generateLocalResources(cobraCmd, flagRemoveJSON, selectedResources, appInfo.ApplicationID); err != nil {
		return errors.WrapError("failed to update project resources", err)
	}

	if flagRemoveJSON {
		return utils.WriteJSON(cobraCmd, map[string]any{
			"resourceId": flagRemoveResourceID,
			"attached":   false,
		})
	}
	return nil
}
