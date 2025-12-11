package app

import (
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/major-technology/cli/clients/git"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
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
		return "", err
	}

	if remoteURL == "" {
		return "", fmt.Errorf("no git remote found in directory")
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return "", errors.WrapError("failed to parse git remote URL", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return "", fmt.Errorf("API client not initialized")
	}

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return "", errors.WrapError("failed to get application", err)
	}

	return appResp.ApplicationID, nil
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

// pullOrCloneWithRetries retries a git clone or pull operation with exponential backoff
func pullOrCloneWithRetries(cmd *cobra.Command, workingDir, sshURL, httpsURL string) error {
	maxRetries := 3
	baseDelay := 200 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1)) // Exponential backoff: 200ms, 400ms, 800ms
			time.Sleep(delay)
		}

		var err error
		if _, statErr := os.Stat(workingDir); statErr == nil {
			// Directory exists, pull
			cmd.Printf("Pulling latest changes...\n")
			err = git.Pull(workingDir)
		} else {
			// Directory doesn't exist, clone
			cmd.Printf("Cloning repository...\n")
			_, err = cloneRepository(sshURL, httpsURL, workingDir)
		}

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
// Returns the path to the generated file and the number of variables written.
func generateEnvFile(targetDir string) (string, int, error) {
	orgID, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return "", 0, errors.WrapError("failed to get default organization", errors.ErrorNoOrganizationSelected)
	}

	applicationID, err := getApplicationIDFromDir(targetDir)
	if err != nil {
		return "", 0, errors.WrapError("failed to get application ID", err)
	}

	apiClient := singletons.GetAPIClient()

	envVars, err := apiClient.GetApplicationEnv(orgID, applicationID)
	if err != nil {
		return "", 0, errors.WrapError("failed to get environment variables", err)
	}

	gitRoot := targetDir
	if gitRoot == "" {
		gitRoot, err = git.GetRepoRoot()
		if err != nil {
			return "", 0, errors.WrapError("failed to get git repository root", err)
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
		return "", 0, errors.WrapError("failed to write .env file", err)
	}

	return envFilePath, len(envVars), nil
}
