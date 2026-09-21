package app

import (
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagInfoJSON bool

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display information about the current application",
	Long:  `Display information about the application in the current directory, including the application ID, deploy status, and URL.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInfo(cmd)
	},
}

func init() {
	infoCmd.Flags().BoolVar(&flagInfoJSON, "json", false, "Output in JSON format")
}

func runInfo(cmd *cobra.Command) error {
	applicationID, err := getApplicationID()
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()
	appInfo, err := apiClient.GetApplicationInfo(applicationID)
	if err != nil {
		return err
	}

	if flagInfoJSON {
		type infoJSON struct {
			ApplicationID   string  `json:"applicationId"`
			Name            string  `json:"name"`
			DeployStatus    string  `json:"deployStatus"`
			AppURL          *string `json:"appUrl"`
			DeployError     *string `json:"deployError"`
			IsPublic        bool    `json:"isPublic"`
			WebhooksEnabled bool    `json:"webhooksEnabled"`
			DeployedHash    *string `json:"deployedHash"`
		}
		return utils.WriteJSON(cmd, infoJSON{
			ApplicationID:   appInfo.ApplicationID,
			Name:            appInfo.Name,
			DeployStatus:    appInfo.DeployStatus,
			AppURL:          appInfo.AppURL,
			DeployError:     appInfo.DeployError,
			IsPublic:        appInfo.IsPublic,
			WebhooksEnabled: appInfo.WebhooksEnabled,
			DeployedHash:    appInfo.DeployedHash,
		})
	}

	cmd.Printf("Application ID: %s\n", appInfo.ApplicationID)
	cmd.Printf("Name: %s\n", appInfo.Name)
	cmd.Printf("Deploy Status: %s\n", appInfo.DeployStatus)

	if appInfo.DeployError != nil && *appInfo.DeployError != "" {
		cmd.Printf("Deploy Error: %s\n", *appInfo.DeployError)
	}

	if appInfo.DeployedHash != nil && *appInfo.DeployedHash != "" {
		cmd.Printf("Deployed Hash: %s\n", *appInfo.DeployedHash)
	}

	cmd.Printf("Public: %t\n", appInfo.IsPublic)
	cmd.Printf("Webhooks Enabled: %t\n", appInfo.WebhooksEnabled)

	if appInfo.AppURL != nil {
		cmd.Printf("URL: %s\n", *appInfo.AppURL)
	}

	return nil
}
