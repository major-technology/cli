package git

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"

	clierrors "github.com/major-technology/cli/errors"
)

// RemoteInfo contains parsed information from a git remote URL
type RemoteInfo struct {
	Owner string
	Repo  string
}

// GetRemoteURL retrieves the git remote URL from the current directory
func GetRemoteURL() (string, error) {
	return GetRemoteURLFromDir("")
}

// GetRemoteURLFromDir retrieves the git remote URL from the specified directory.
// If dir is empty, it uses the current directory.
func GetRemoteURLFromDir(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.Output()
	if err != nil {
		// Check if this is a "not a git repository" error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "not a git repository") {
				return "", clierrors.ErrorNotGitRepository
			}
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Clone clones a git repository
func Clone(url, targetDir string) error {
	cmd := exec.Command("git", "clone", url, targetDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include the git output in the error message
		return clierrors.WrapError("git clone failed: "+string(output), err)
	}
	return nil
}

// RemoveRemote removes a git remote
func RemoveRemote(repoDir, remoteName string) error {
	cmd := exec.Command("git", "remote", "remove", remoteName)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// AddRemote adds a git remote
func AddRemote(repoDir, remoteName, url string) error {
	cmd := exec.Command("git", "remote", "add", remoteName, url)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Push pushes to the remote repository
func Push(repoDir string) error {
	cmd := exec.Command("git", "push", "--force", "-u", "origin", "main")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// ParseRemoteURL parses a git remote URL and extracts the owner and repository name
// Supports formats:
// - SSH: git@github.com:owner/repo.git
// - HTTPS: https://github.com/owner/repo.git
// - HTTPS (no .git): https://github.com/owner/repo
func ParseRemoteURL(remoteURL string) (*RemoteInfo, error) {
	remoteURL = strings.TrimSpace(remoteURL)

	// SSH format: git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); matches != nil {
		return &RemoteInfo{
			Owner: matches[1],
			Repo:  strings.TrimSuffix(matches[2], ".git"),
		}, nil
	}

	// HTTPS format: https://github.com/owner/repo.git or https://github.com/owner/repo
	httpsPattern := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(remoteURL); matches != nil {
		return &RemoteInfo{
			Owner: matches[1],
			Repo:  strings.TrimSuffix(matches[2], ".git"),
		}, nil
	}

	return nil, clierrors.ErrorUnsupportedGitRemoteURLWithFormat(remoteURL)
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// IsGitRepository checks if the current directory is a git repository
func IsGitRepository() bool {
	return IsGitRepositoryDir("")
}

// IsGitRepositoryDir checks if the specified directory is a git repository.
// If dir is empty, it uses the current directory.
func IsGitRepositoryDir(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if dir != "" {
		cmd.Dir = dir
	}
	err := cmd.Run()
	return err == nil
}

// InitRepository initializes a new git repository in the specified directory.
// If dir is empty, it uses the current directory.
func InitRepository(dir string) error {
	cmd := exec.Command("git", "init")
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return clierrors.WrapError("git init failed: "+string(output), err)
	}
	return nil
}

// SetRemoteURL sets or updates the origin remote URL.
// If the remote doesn't exist, it adds it. If it exists, it updates it.
func SetRemoteURL(dir, remoteName, url string) error {
	// First try to set the URL (works if remote exists)
	cmd := exec.Command("git", "remote", "set-url", remoteName, url)
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Run(); err != nil {
		// Remote doesn't exist, add it
		cmd = exec.Command("git", "remote", "add", remoteName, url)
		if dir != "" {
			cmd.Dir = dir
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			return clierrors.WrapError("failed to add remote: "+string(output), err)
		}
	}
	return nil
}

// HasUncommittedChanges checks if there are uncommitted changes in the repository
func HasUncommittedChanges() (bool, error) {
	// Check for staged and unstaged changes
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// Add stages all changes
func Add() error {
	cmd := exec.Command("git", "add", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Commit commits staged changes with the given message
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PushToMain pushes commits to the remote repository on main branch
func PushToMain() error {
	cmd := exec.Command("git", "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Pull pulls the latest changes from the remote repository
func Pull(repoDir string) error {
	cmd := exec.Command("git", "pull")
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include the git output in the error message
		return clierrors.WrapError("git pull failed: "+string(output), err)
	}
	return nil
}

// GetCurrentGithubUser attempts to retrieve the GitHub username of the current user
// by checking SSH authentication and git configuration.
func GetCurrentGithubUser() (string, error) {
	// 1. Try SSH authentication
	// ssh -T -o BatchMode=yes -o ConnectTimeout=2 git@github.com
	// This usually returns exit code 1 on success with "Hi <username>! ..."
	cmd := exec.Command("ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=2", "git@github.com")
	output, _ := cmd.CombinedOutput() // We expect an error (exit code 1), so we ignore it and parse output

	outputStr := string(output)
	// Regex matches: Hi <username>! You've successfully authenticated...
	// GitHub usernames are alphanumeric with single hyphens, max 39 chars
	sshPattern := regexp.MustCompile(`Hi ([a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)! You've successfully authenticated`)
	if matches := sshPattern.FindStringSubmatch(outputStr); len(matches) > 1 {
		return matches[1], nil
	}

	// 2. Check git config for github.user
	cmd = exec.Command("git", "config", "--get", "github.user")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output)), nil
	}

	// 3. Check git config user.email for GitHub noreply address
	cmd = exec.Command("git", "config", "--get", "user.email")
	output, err = cmd.Output()
	if err == nil {
		email := strings.TrimSpace(string(output))
		// Matches: <id>+<username>@users.noreply.github.com
		// Example: 123456+jasonbao@users.noreply.github.com
		emailPattern := regexp.MustCompile(`^(?:(\d+)\+)?([^@]+)@users\.noreply\.github\.com$`)
		if matches := emailPattern.FindStringSubmatch(email); len(matches) > 2 {
			// matches[2] contains the username part
			return matches[2], nil
		}
	}

	return "", nil
}
