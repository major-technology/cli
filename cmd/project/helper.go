package project

import (
	stderrors "errors"
	"net/http"

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
		if isProjectNotFoundError(err) {
			return "", "", errors.ErrorProjectNotFound
		}
		return "", "", errors.WrapError("failed to get project", err)
	}

	if resp.ProjectID == "" {
		return "", "", errors.ErrorProjectNotFound
	}

	return resp.ProjectID, resp.OrganizationID, nil
}

// isProjectNotFoundError reports whether err is the API's HTTP 404 for
// GetProjectByRepo - the server's signal that no project is registered for
// this repository (see apps/api's projects service: getProjectByRepo throws
// a generic NotFoundError, not a project-specific error code, so the CLI's
// error-code mapping falls through to the unmapped-code path where the
// originating status is still available on the CLIError).
func isProjectNotFoundError(err error) bool {
	var cliErr *errors.CLIError
	return stderrors.As(err, &cliErr) && cliErr.StatusCode == http.StatusNotFound
}
