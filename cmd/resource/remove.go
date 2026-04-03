package resource

import (
	"fmt"
	"os"
	"os/exec"

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

	// Get current app resources
	appResources, err := apiClient.GetApplicationResources(appInfo.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to get application resources", err)
	}

	// Find the resource and build filtered list
	var removedName string
	var removedType string
	resourceIDs := make([]string, 0, len(appResources.Resources))
	for _, r := range appResources.Resources {
		if r.ID == flagRemoveResourceID {
			removedName = r.Name
			removedType = r.Type
			continue
		}
		resourceIDs = append(resourceIDs, r.ID)
	}

	if removedName == "" {
		return fmt.Errorf("resource with ID %q is not attached to this application", flagRemoveResourceID)
	}

	// Save the updated resource list
	_, err = apiClient.SaveApplicationResources(appInfo.OrganizationID, appInfo.ApplicationID, resourceIDs)
	if err != nil {
		return errors.WrapError("failed to save resources", err)
	}

	// Remove local client code
	cobraCmd.Printf("Removing resource: %s (%s)...\n", removedName, removedType)
	framework := utils.DetectFramework(".")
	args := []string{"exec", "major-client", "remove", removedName}
	if framework != "" {
		args = append(args, "--framework", framework)
	}
	pnpmCmd := exec.Command("pnpm", args...)
	pnpmCmd.Stdout = os.Stdout
	pnpmCmd.Stderr = os.Stderr
	if err := pnpmCmd.Run(); err != nil {
		cobraCmd.Printf("Warning: Failed to remove local resource files: %v\n", err)
	}

	cobraCmd.Printf("Resource %q removed successfully.\n", removedName)
	return nil
}
