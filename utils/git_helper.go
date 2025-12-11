package utils

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
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

// CanUseSSH checks if SSH is available and configured for git
func CanUseSSH() bool {
	// Test actual SSH connectivity to GitHub
	// ssh -T returns exit code 1 even on success (no shell access), so we check output
	cmd := exec.Command("ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "git@github.com")
	output, _ := cmd.CombinedOutput() // Ignore error since exit code 1 is expected on success

	// GitHub returns "Hi <username>! You've successfully authenticated..." on success
	return strings.Contains(string(output), "successfully authenticated")
}

// CheckRepositoryAccess attempts to check if a repository is accessible via git ls-remote
// Returns true if accessible, false otherwise
func CheckRepositoryAccess(sshURL, httpsURL string) bool {
	// Try SSH first if available
	if CanUseSSH() && sshURL != "" {
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

// ExtractGitHubURL extracts the GitHub repository URL from SSH or HTTPS clone URL
// Returns format: https://github.com/<owner>/<repo>
func ExtractGitHubURL(cloneURL string) (string, error) {
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

// EnsureRepositoryAccess ensures the user has access to the repository by inviting them as a collaborator
// This function prompts for GitHub username, sends an invite, and waits for access to be granted
func EnsureRepositoryAccess(cmd *cobra.Command, appID string, sshURL string, httpsURL string) error {
	// First check if the user already has access to the repository
	if CheckRepositoryAccess(sshURL, httpsURL) {
		cmd.Println("✓ You already have access to the repository")
		return nil
	}

	// Check if GitHub username is stored in keychain
	storedUsername, err := mjrToken.GetGithubUsername()
	if err != nil {
		return errors.WrapError("failed to check stored GitHub username", err)
	}

	// If not in keychain, try to get it from git config/ssh
	if storedUsername == "" {
		gitUsername, err := git.GetCurrentGithubUser()
		if err == nil && gitUsername != "" {
			storedUsername = gitUsername
		}
	}

	var githubUsername string

	// If we have a stored username (from keychain or git), confirm with the user
	if storedUsername != "" {
		var useStored bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Use GitHub username: %s?", storedUsername)).
					Description("We found this GitHub username. Would you like to use it?").
					Value(&useStored),
			),
		)

		if err := confirmForm.Run(); err != nil {
			return errors.WrapError("failed to confirm GitHub username", err)
		}

		if useStored {
			githubUsername = storedUsername
		}
	}

	if githubUsername == "" {
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
			return errors.WrapError("failed to get GitHub username", err)
		}

		if err := mjrToken.StoreGithubUsername(githubUsername); err != nil {
			return errors.WrapError("failed to save GitHub username", err)
		}
	}

	cmd.Printf("\nAdding @%s as a collaborator to the repository...\n", githubUsername)

	// Get API client
	apiClient := singletons.GetAPIClient()

	// Add user as GitHub collaborator
	_, err = apiClient.AddGithubCollaborators(appID, githubUsername)
	if err != nil {
		return errors.WrapError("failed to add GitHub collaborator", err)
	}

	cmd.Println("✓ Invitation sent!")

	// Try to extract and open the GitHub repository URL
	cloneURL := httpsURL
	if cloneURL == "" {
		cloneURL = sshURL
	}

	githubURL, urlErr := ExtractGitHubURL(cloneURL)
	if urlErr == nil {
		cmd.Printf("\nPlease accept the invitation at: %s\n", githubURL)
		_ = OpenBrowser(githubURL)
	}

	// Poll for repository access
	if !pollForRepositoryAccess(cmd, sshURL, httpsURL) {
		return errors.ErrorRepositoryAccessTimeout
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
			if CheckRepositoryAccess(sshURL, httpsURL) {
				return true
			}
			cmd.Print(".")
		}
	}
}
