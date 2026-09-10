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

var (
	flagAddResourceID     string
	flagAddJSON           bool
	addResourcesToProject = utils.AddResourcesToProject
)

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
	addCmd.Flags().BoolVar(&flagAddJSON, "json", false, "Output in JSON format")
	addCmd.MarkFlagRequired("id")
}

func runAdd(cobraCmd *cobra.Command) error {
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	apiClient := singletons.GetAPIClient()

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

	existingResources, err := utils.ReadLocalResources(".")
	if err != nil {
		fmt.Fprintf(cobraCmd.ErrOrStderr(), "Warning: Could not read existing resources: %v\n", err)
		existingResources = []utils.LocalResource{}
	}

	selectedIDs := make([]string, 0, len(existingResources)+1)
	for _, r := range existingResources {
		selectedIDs = append(selectedIDs, r.ID)
	}
	selectedIDs = append(selectedIDs, flagAddResourceID)

	_, err = apiClient.SaveApplicationResources(appInfo.OrganizationID, appInfo.ApplicationID, selectedIDs)
	if err != nil {
		return errors.WrapError("failed to save resources", err)
	}

	selectedResources := utils.ResolveResourceItems(selectedIDs, orgResources.Resources)
	if err := generateLocalResources(cobraCmd, flagAddJSON, selectedResources, appInfo.ApplicationID); err != nil {
		return errors.WrapError("failed to add resource to project", err)
	}

	if flagAddJSON {
		return utils.WriteJSON(cobraCmd, map[string]any{
			"resourceId": flagAddResourceID,
			"attached":   true,
		})
	}
	return nil
}

func generateLocalResources(cmd *cobra.Command, jsonOut bool, resources []api.ResourceItem, applicationID string) error {
	restore := routeJSONChatter(cmd, jsonOut)
	defer restore()
	return addResourcesToProject(cmd, ".", resources, applicationID)
}

func routeJSONChatter(cmd *cobra.Command, jsonOut bool) func() {
	if !jsonOut {
		return func() {}
	}
	stdout := cmd.OutOrStdout()
	cmd.SetOut(cmd.ErrOrStderr())
	return func() {
		cmd.SetOut(stdout)
	}
}
