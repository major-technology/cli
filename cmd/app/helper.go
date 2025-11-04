package app

import (
	"fmt"

	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/singletons"
)

// getApplicationID retrieves the application ID for the current git repository
func getApplicationID() (string, error) {
	return getApplicationIDFromDir("")
}

// getApplicationIDFromDir retrieves the application ID for a git repository in the specified directory.
// If dir is empty, it uses the current directory.
func getApplicationIDFromDir(dir string) (string, error) {
	// Get the git remote URL from the specified directory
	remoteURL, err := git.GetRemoteURLFromDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get git remote: %w", err)
	}

	if remoteURL == "" {
		return "", fmt.Errorf("no git remote found in directory")
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse git remote URL: %w", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return "", fmt.Errorf("API client not initialized")
	}

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return "", fmt.Errorf("failed to get application: %w", err)
	}

	return appResp.ApplicationID, nil
}
