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

	// Verify the resource exists in the org
	orgResources, err := apiClient.GetResources(appInfo.OrganizationID)
	if err != nil {
		return errors.WrapError("failed to get resources", err)
	}

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

	// Get current app resources
	appResources, err := apiClient.GetApplicationResources(appInfo.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to get application resources", err)
	}

	// Check if already attached
	resourceIDs := make([]string, 0, len(appResources.Resources)+1)
	for _, r := range appResources.Resources {
		if r.ID == flagAddResourceID {
			cobraCmd.Printf("Resource %q is already attached to this application.\n", targetResource.Name)
			return nil
		}
		resourceIDs = append(resourceIDs, r.ID)
	}

	// Add the new resource
	resourceIDs = append(resourceIDs, flagAddResourceID)

	_, err = apiClient.SaveApplicationResources(appInfo.OrganizationID, appInfo.ApplicationID, resourceIDs)
	if err != nil {
		return errors.WrapError("failed to save resources", err)
	}

	// Generate local client code — pass full resource list so the diff works correctly
	allResources := append(appResources.Resources, *targetResource)
	if err := utils.AddResourcesToProject(cobraCmd, ".", allResources, appInfo.ApplicationID); err != nil {
		return errors.WrapError("failed to add resource to project", err)
	}

	cobraCmd.Printf("Resource %q added successfully.\n", targetResource.Name)
	return nil
}
