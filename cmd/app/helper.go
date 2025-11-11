package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/git"
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

// ensureRepositoryAccess ensures the user has access to the repository by inviting them as a collaborator
// This function prompts for GitHub username, sends an invite, and waits for access to be granted
func ensureRepositoryAccess(cmd *cobra.Command, appID string, sshURL string, httpsURL string) error {
	// Prompt for GitHub username
	var githubUsername string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("What is your GitHub username?").
				Value(&githubUsername).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("GitHub username is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	cmd.Printf("\nAdding @%s as a collaborator to the repository...\n", githubUsername)

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	// Add user as GitHub collaborator
	_, err := apiClient.AddGithubCollaborators(appID, githubUsername)
	if err != nil {
		return fmt.Errorf("failed to add GitHub collaborator: %w", err)
	}

	cmd.Println("✓ Invitation sent!")

	// Try to extract and open the GitHub repository URL
	cloneURL := httpsURL
	if cloneURL == "" {
		cloneURL = sshURL
	}

	githubURL, urlErr := extractGitHubURL(cloneURL)
	if urlErr == nil {
		cmd.Printf("\nPlease accept the invitation at: %s\n", githubURL)
		_ = utils.OpenBrowser(githubURL)
		cmd.Printf("You may need to refresh the page to see the invitation.\n")
	}

	// Poll for repository access
	if !pollForRepositoryAccess(cmd, sshURL, httpsURL) {
		return fmt.Errorf("timeout waiting for repository access - please try again after accepting the invitation")
	}

	cmd.Println("\n✓ Repository access granted!")
	return nil
}

// pollForRepositoryAccess polls the repository to check if access has been granted
// Polls every 2 seconds with a 5 minute timeout
// Returns true if access is granted, false if timeout
func pollForRepositoryAccess(cmd *cobra.Command, sshURL, httpsURL string) bool {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			if checkRepositoryAccess(sshURL, httpsURL) {
				return true
			}
			cmd.Print(".")
		}
	}
}

// retryGitOperation retries a git clone or pull operation with exponential backoff
// This handles race conditions where permissions take a moment to propagate after being granted
func retryGitOperation(cmd *cobra.Command, workingDir, sshURL, httpsURL string) error {
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

	return fmt.Errorf("repository access not available after %d attempts", maxRetries)
}
