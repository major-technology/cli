package app

import (
	"fmt"
	"os/exec"
	"strings"

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
		return "", err
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

// canUseSSH checks if SSH is available and configured for git
func canUseSSH() bool {
	// Check if ssh-agent is running and has keys
	cmd := exec.Command("ssh-add", "-l")
	err := cmd.Run()
	return err == nil
}

// cloneRepository clones a repository using SSH or HTTPS based on availability
// Returns the clone method used ("SSH" or "HTTPS") and any error
func cloneRepository(sshURL, httpsURL, targetDir string) (string, error) {
	// Determine which clone URL to use
	useSSH := false
	if canUseSSH() && sshURL != "" {
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
		return "", fmt.Errorf("failed to clone repository using %s: %w", cloneMethod, err)
	}

	return cloneMethod, nil
}

// isGitAuthError checks if the error is related to git authentication/permission issues
func isGitAuthError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

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

	for _, pattern := range authErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// extractGitHubURL extracts the GitHub repository URL from SSH or HTTPS clone URL
// Returns format: https://github.com/<owner>/<repo>
func extractGitHubURL(cloneURL string) (string, error) {
	if cloneURL == "" {
		return "", fmt.Errorf("clone URL is empty")
	}

	// Parse the URL to get owner and repo
	remoteInfo, err := git.ParseRemoteURL(cloneURL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://github.com/%s/%s", remoteInfo.Owner, remoteInfo.Repo), nil
}

// checkRepositoryAccess attempts to check if a repository is accessible via git ls-remote
// Returns true if accessible, false otherwise
func checkRepositoryAccess(sshURL, httpsURL string) bool {
	// Try SSH first if available
	if canUseSSH() && sshURL != "" {
		return testGitAccess(sshURL)
	}

	// Fall back to HTTPS
	if httpsURL != "" {
		if testGitAccess(httpsURL) {
			return true
		}
	}

	return false
}

// testGitAccess tests if a git repository is accessible using git ls-remote
func testGitAccess(repoURL string) bool {
	cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
	// Suppress output
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
}
