package git

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Clone clones a git repository
func Clone(url, targetDir string) error {
	cmd := exec.Command("git", "clone", url, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

	return nil, fmt.Errorf("unsupported remote URL format: %s", remoteURL)
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
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
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
