package app

import (
	"encoding/json"
	"fmt"

	"github.com/major-technology/cli/singletons"
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
	// Get application ID
	applicationID, err := getApplicationID()
	if err != nil {
		return err
	}

	// Try to get extended info from the new endpoint
	apiClient := singletons.GetAPIClient()
	appInfo, err := apiClient.GetApplicationInfo(applicationID)
	if err != nil {
		// Graceful fallback: if endpoint doesn't exist yet, just show app ID
		cmd.Printf("Application ID: %s\n", applicationID)
		return nil
	}

	if flagInfoJSON {
		type infoJSON struct {
			ApplicationID string  `json:"applicationId"`
			Name          string  `json:"name"`
			DeployStatus  string  `json:"deployStatus"`
			AppURL        *string `json:"appUrl"`
		}

		data, err := json.Marshal(infoJSON{
			ApplicationID: appInfo.ApplicationID,
			Name:          appInfo.Name,
			DeployStatus:  appInfo.DeployStatus,
			AppURL:        appInfo.AppURL,
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	cmd.Printf("Application ID: %s\n", appInfo.ApplicationID)
	cmd.Printf("Name: %s\n", appInfo.Name)
	cmd.Printf("Deploy Status: %s\n", appInfo.DeployStatus)
	if appInfo.AppURL != nil {
		cmd.Printf("URL: %s\n", *appInfo.AppURL)
	}

	return nil
}
