package utils

import (
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/errors"
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
		return nil, errors.ErrorNoGitRemoteFoundInDirectory
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return nil, errors.WrapError("failed to parse git remote URL", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return nil, errors.WrapError("failed to get application", err)
	}

	return appResp, nil
}
