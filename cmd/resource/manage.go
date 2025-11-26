package resource

import (
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// manageCmd represents the manage command
var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Manage application resources",
	Long:  `Select and configure resources for your application.`,
	PreRunE: middleware.ChainParent(
		middleware.CheckLogin,
		middleware.CheckNodeInstalled,
		middleware.CheckNodeVersion("22.12"),
		middleware.CheckPnpmInstalled,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runManage(cobraCmd)
	},
}

func runManage(cobraCmd *cobra.Command) error {
	// Get application info from current directory
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	apiClient := singletons.GetAPIClient()

	cobraCmd.Println("\nSelecting resources for your application...")
	selectedResources, err := utils.SelectApplicationResources(cobraCmd, apiClient, appInfo.OrganizationID, appInfo.ApplicationID)
	if err != nil {
		return errors.ErrorFailedToSelectResourcesTryAgain
	}

	if appInfo.TemplateName == nil {
		return errors.ErrorOldProjectNotSupported
	}

	templateName := *appInfo.TemplateName

	if err := utils.AddResourcesToProject(cobraCmd, ".", selectedResources, appInfo.ApplicationID, templateName); err != nil {
		return errors.ErrorFailedToSelectResourcesTryAgain
	}

	return nil
}

func init() {
	// Add manage subcommand
	Cmd.AddCommand(manageCmd)
}
