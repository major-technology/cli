package appclient

import (
	"fmt"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagAddID   string
	flagAddName string
	flagAddJSON bool
	// ensurePackage is a variable so tests skip pnpm.
	ensurePackage = ensureAppClientPackage
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a client for calling another application",
	Long:  `Generate a fetch client for another application in this organization. Use 'major app list' to find application IDs. The deploy grants access from the generated client.`,
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
	addCmd.Flags().StringVar(&flagAddID, "id", "", "Application ID to call")
	addCmd.Flags().StringVar(&flagAddName, "name", "", "Client name (defaults to the application name)")
	addCmd.Flags().BoolVar(&flagAddJSON, "json", false, "Output in JSON format")
	addCmd.MarkFlagRequired("id")
}

func runAdd(cobraCmd *cobra.Command) error {
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	if flagAddID == appInfo.ApplicationID {
		return fmt.Errorf("an application cannot add a client for itself")
	}

	apps, err := singletons.GetAPIClient().GetOrganizationApplications(appInfo.OrganizationID)
	if err != nil {
		return errors.WrapError("failed to list applications", err)
	}

	name := ""
	for _, a := range apps.Applications {
		if a.ID == flagAddID {
			name = a.Name
			break
		}
	}

	if name == "" {
		return fmt.Errorf("application with ID %q not found in organization", flagAddID)
	}

	if flagAddName != "" {
		name = flagAddName
	}

	if err := ensurePackage(cobraCmd, "."); err != nil {
		return err
	}

	if err := runAppClientCLI(cobraCmd, ".", "add", flagAddID, name); err != nil {
		return err
	}

	if flagAddJSON {
		return utils.WriteJSON(cobraCmd, map[string]any{"appId": flagAddID, "name": name, "added": true})
	}

	cobraCmd.Printf("Added client for %s (%s)\n", name, flagAddID)
	return nil
}
