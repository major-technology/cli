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
	cmd := exec.Command("git", "remote", "get-url", "origin")
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
