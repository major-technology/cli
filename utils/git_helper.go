package utils

import (
	"fmt"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/singletons"
)

// GetApplicationID retrieves the application ID for the current git repository
func GetApplicationID() (string, error) {
	info, err := GetApplicationInfo("")
	if err != nil {
		return "", err
	}
	return info.ApplicationID, nil
}

// GetApplicationIDFromDir retrieves the application ID and organization ID for a git repository in the specified directory.
// If dir is empty, it uses the current directory.
// Deprecated: Use GetApplicationInfo instead
func GetApplicationIDFromDir(dir string) (string, string, error) {
	info, err := GetApplicationInfo(dir)
	if err != nil {
		return "", "", err
	}
	return info.ApplicationID, info.OrganizationID, nil
}

// GetApplicationInfo retrieves full application information for a git repository in the specified directory.
// If dir is empty, it uses the current directory.
func GetApplicationInfo(dir string) (*api.GetApplicationByRepoResponse, error) {
	// Get the git remote URL from the specified directory
	remoteURL, err := git.GetRemoteURLFromDir(dir)
	if err != nil {
		return nil, err
	}

	if remoteURL == "" {
		return nil, fmt.Errorf("no git remote found in directory")
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse git remote URL: %w", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return nil, fmt.Errorf("API client not initialized")
	}

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	return appResp, nil
}
