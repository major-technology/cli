package app

import (
	"fmt"

	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display information about the current application",
	Long:  `Display information about the application in the current directory, including the application ID.`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runInfo(cmd))
	},
}

func runInfo(cmd *cobra.Command) error {
	// Get the git remote URL from the current directory
	remoteURL, err := git.GetRemoteURL()
	if err != nil {
		return fmt.Errorf("failed to get git remote: %w", err)
	}

	if remoteURL == "" {
		return fmt.Errorf("no git remote found in current directory")
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse git remote URL: %w", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	// Print only the application ID
	cmd.Printf("Application ID: %s\n", appResp.ApplicationID)

	return nil
}
