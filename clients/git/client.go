package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	clierrors "github.com/major-technology/cli/errors"
)

var nonInteractive bool

// SetNonInteractive controls whether remote git subprocesses disable terminal
// credential prompts. HTTPS credential helpers are left enabled.
func SetNonInteractive(v bool) {
	nonInteractive = v
}

func applyNonInteractiveGit(cmd *exec.Cmd) {
	if !nonInteractive {
		return
	}
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	env = append(env, "GIT_SSH_COMMAND="+ensureSSHBatchMode(os.Getenv("GIT_SSH_COMMAND")))
	cmd.Env = env
	cmd.Stdin = nil
}

// ensureSSHBatchMode returns an SSH command with a single BatchMode=yes option.
// Substrings such as a key path containing "BatchMode=yes" are not treated as
// options. Conflicting BatchMode values are removed and replaced.
func ensureSSHBatchMode(sshCmd string) string {
	if strings.TrimSpace(sshCmd) == "" {
		sshCmd = "ssh"
	}
	tokens := splitGitSSHCommand(sshCmd)
	out := make([]string, 0, len(tokens)+2)
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "-o" {
			if i+1 >= len(tokens) {
				out = append(out, tok)
				break
			}
			opt := tokens[i+1]
			if sshOptionKey(opt) == "batchmode" {
				i++
				continue
			}
			if strings.EqualFold(opt, "BatchMode") {
				i++
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					i++
				}
				continue
			}
			out = append(out, tok, opt)
			i++
			continue
		}
		if len(tok) > 2 && strings.HasPrefix(tok, "-o") && sshOptionKey(tok[2:]) == "batchmode" {
			continue
		}
		out = append(out, tok)
	}
	out = append(out, "-o", "BatchMode=yes")
	return strings.Join(out, " ")
}

func sshOptionKey(opt string) string {
	key, _, ok := strings.Cut(opt, "=")
	if !ok {
		return ""
	}
	return strings.ToLower(key)
}

func splitGitSSHCommand(s string) []string {
	var tokens []string
	var b strings.Builder
	quote := rune(0)
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t':
			if b.Len() > 0 {
				tokens = append(tokens, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens
}

// ConfigureRemoteCommand applies non-interactive git environment to a command.
func ConfigureRemoteCommand(cmd *exec.Cmd) {
	applyNonInteractiveGit(cmd)
}

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
	applyNonInteractiveGit(cmd)
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
	applyNonInteractiveGit(cmd)
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
	applyNonInteractiveGit(cmd)
	return cmd.Run()
}

// Pull pulls the latest changes from the remote repository
func Pull(repoDir string) error {
	cmd := exec.Command("git", "pull")
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	applyNonInteractiveGit(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include the git output in the error message
		return clierrors.WrapError("git pull failed: "+string(output), err)
	}
	return nil
}

// IsBehindRemote checks if the local branch is behind origin/main.
// Returns whether it's behind, how many commits behind, and any error.
// Uses a 5-second timeout to avoid blocking if the network is unavailable.
func IsBehindRemote() (bool, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch latest from origin
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", "main", "--quiet")
	applyNonInteractiveGit(fetchCmd)
	if err := fetchCmd.Run(); err != nil {
		return false, 0, err
	}

	// Count commits local is behind
	revListCmd := exec.Command("git", "rev-list", "--count", "HEAD..origin/main")
	output, err := revListCmd.Output()
	if err != nil {
		return false, 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return false, 0, err
	}

	return count > 0, count, nil
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
