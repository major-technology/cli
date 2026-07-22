package project

import (
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
)

// getProjectAndOrgID resolves the project for the current directory from the
// git remote origin URL, mirroring how cmd/app resolves applications.
func getProjectAndOrgID() (string, string, error) {
	remoteURL, err := git.GetRemoteURLFromDir("")
	if err != nil {
		return "", "", errors.ErrorNotInProjectDirectory
	}

	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return "", "", errors.WrapError("failed to parse git remote URL", err)
	}

	apiClient := singletons.GetAPIClient()

	resp, err := apiClient.GetProjectByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return "", "", errors.WrapError("failed to get project", err)
	}

	if resp.ProjectID == "" {
		return "", "", errors.ErrorProjectNotFound
	}

	return resp.ProjectID, resp.OrganizationID, nil
}
