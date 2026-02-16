package app

import (
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// getApplicationID retrieves the application ID for the current git repository
func getApplicationID() (string, error) {
	appID, _, err := getApplicationAndOrgIDFromDir("")
	return appID, err
}

// getApplicationAndOrgID retrieves the application ID and organization ID for the current git repository
func getApplicationAndOrgID() (string, string, error) {
	return getApplicationAndOrgIDFromDir("")
}

// getApplicationIDFromDir retrieves the application ID for a git repository in the specified directory.
// If dir is empty, it uses the current directory.
func getApplicationIDFromDir(dir string) (string, error) {
	appID, _, err := getApplicationAndOrgIDFromDir(dir)
	return appID, err
}

// getApplicationAndOrgIDFromDir retrieves the application ID and organization ID for a git repository in the specified directory.
// If dir is empty, it uses the current directory.
func getApplicationAndOrgIDFromDir(dir string) (string, string, error) {
	// Get the git remote URL from the specified directory
	remoteURL, err := git.GetRemoteURLFromDir(dir)
	if err != nil {
		return "", "", err
	}

	if remoteURL == "" {
		return "", "", fmt.Errorf("no git remote found in directory")
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return "", "", errors.WrapError("failed to parse git remote URL", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return "", "", fmt.Errorf("API client not initialized")
	}

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return "", "", errors.WrapError("failed to get application", err)
	}

	return appResp.ApplicationID, appResp.OrganizationID, nil
}

// getPreferredCloneURL returns the preferred clone URL based on SSH availability
func getPreferredCloneURL(sshURL, httpsURL string) (url string, method string, err error) {
	if utils.CanUseSSH() && sshURL != "" {
		return sshURL, "SSH", nil
	}
	if httpsURL != "" {
		return httpsURL, "HTTPS", nil
	}
	return "", "", fmt.Errorf("no valid clone method available")
}

// ensureGitRepository ensures a directory is a properly configured git repository.
// If the directory doesn't exist, it clones the repo.
// If it exists but isn't a git repo, it initializes git and sets origin.
// If it exists and is a git repo, it ensures origin is set correctly and pulls.
// Returns the working directory path and any error.
func ensureGitRepository(cmd *cobra.Command, targetDir, sshURL, httpsURL string) error {
	cloneURL, cloneMethod, err := getPreferredCloneURL(sshURL, httpsURL)
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// Directory doesn't exist - clone fresh
		cmd.Printf("Cloning repository to '%s' using %s...\n", targetDir, cloneMethod)
		return git.Clone(cloneURL, targetDir)
	}

	// Directory exists - check if it's a git repo
	if !git.IsGitRepositoryDir(targetDir) {
		// Not a git repo - initialize and set origin
		cmd.Printf("Directory '%s' exists but is not a git repository. Initializing...\n", targetDir)
		if err := git.InitRepository(targetDir); err != nil {
			return errors.WrapError("failed to initialize git repository", err)
		}
	}

	// Ensure origin is set correctly
	cmd.Printf("Ensuring git origin is configured correctly...\n")
	if err := git.SetRemoteURL(targetDir, "origin", cloneURL); err != nil {
		return errors.WrapError("failed to set git origin", err)
	}

	// Pull latest changes
	cmd.Printf("Pulling latest changes...\n")
	return git.Pull(targetDir)
}

// cloneRepository clones a repository using SSH or HTTPS based on availability
// Returns the clone method used ("SSH" or "HTTPS") and any error
func cloneRepository(sshURL, httpsURL, targetDir string) (string, error) {
	// Determine which clone URL to use
	useSSH := false
	if utils.CanUseSSH() && sshURL != "" {
		useSSH = true
	} else if httpsURL == "" {
		return "", fmt.Errorf("no valid clone method available")
	}

	cloneURL := httpsURL
	cloneMethod := "HTTPS"
	if useSSH {
		cloneURL = sshURL
		cloneMethod = "SSH"
	}

	// Clone the repository
	if err := git.Clone(cloneURL, targetDir); err != nil {
		return "", errors.WrapError("failed to clone repository using "+cloneMethod, err)
	}

	return cloneMethod, nil
}

// isGitAuthError checks if the error is related to git authentication/permission issues
// It checks all wrapped errors in the chain, not just the top-level error
func isGitAuthError(err error) bool {
	if err == nil {
		return false
	}

	// Common git authentication error patterns
	authErrorPatterns := []string{
		"repository not found", // Catches "ERROR: Repository not found."
		"could not read from remote repository",
		"authentication failed",
		"permission denied",
		"403",
		"401",
		"access denied",
		"fatal: unable to access",
	}

	// Check all errors in the chain
	for e := err; e != nil; e = stderrors.Unwrap(e) {
		errMsg := strings.ToLower(e.Error())
		for _, pattern := range authErrorPatterns {
			if strings.Contains(errMsg, pattern) {
				return true
			}
		}
	}

	return false
}

// ensureGitRepositoryWithRetries retries ensureGitRepository with exponential backoff
func ensureGitRepositoryWithRetries(cmd *cobra.Command, workingDir, sshURL, httpsURL string) error {
	maxRetries := 3
	baseDelay := 200 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1)) // Exponential backoff: 2s, 4s, 8s, 16s, 32s
			cmd.Printf("Waiting %v for GitHub permissions to propagate...\n", delay)
			time.Sleep(delay)
		}

		err := ensureGitRepository(cmd, workingDir, sshURL, httpsURL)
		if err == nil {
			return nil
		}

		// If it's still an auth error, continue retrying
		if !isGitAuthError(err) {
			// Different error type, return immediately
			return err
		}
	}

	return errors.ErrorGitRepositoryAccessFailed
}

// generateEnvFile generates a .env file for the application in the specified directory.
// If targetDir is empty, it uses the current git repository root.
// Returns the path to the generated file and the env vars map.
func generateEnvFile(targetDir string) (string, map[string]string, error) {
	applicationID, orgID, err := getApplicationAndOrgIDFromDir(targetDir)
	if err != nil {
		return "", nil, errors.WrapError("failed to get application ID", err)
	}

	apiClient := singletons.GetAPIClient()

	envVars, err := apiClient.GetApplicationEnv(orgID, applicationID)
	if err != nil {
		return "", nil, errors.WrapError("failed to get environment variables", err)
	}

	gitRoot := targetDir
	if gitRoot == "" {
		gitRoot, err = git.GetRepoRoot()
		if err != nil {
			return "", nil, errors.WrapError("failed to get git repository root", err)
		}
	}

	// Create .env file path
	envFilePath := filepath.Join(gitRoot, ".env")

	// Build the .env file content
	var envContent strings.Builder
	for key, value := range envVars {
		envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Write to .env file
	err = os.WriteFile(envFilePath, []byte(envContent.String()), 0644)
	if err != nil {
		return "", nil, errors.WrapError("failed to write .env file", err)
	}

	return envFilePath, envVars, nil
}
