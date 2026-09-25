package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func applyNonInteractiveGit(cmd *exec.Cmd) error {
	if !nonInteractive {
		return nil
	}
	sshCmd, err := ensureSSHBatchMode(os.Getenv("GIT_SSH_COMMAND"))
	if err != nil {
		return err
	}
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	env = append(env, "GIT_SSH_COMMAND="+sshCmd)
	cmd.Env = env
	cmd.Stdin = nil
	return nil
}

// ensureSSHBatchMode preserves the original GIT_SSH_COMMAND text and inserts
// `-o BatchMode=yes` immediately after a single direct OpenSSH executable so
// OpenSSH's first-wins option order cannot be overridden by a later
// BatchMode=no. Quoted and escaped arguments are left untouched.
//
// Supported: empty (defaults to ssh) or one direct ssh invocation, including a
// path whose basename is ssh. Rejected: env prefixes, wrappers, other
// executables, and compound shell forms (;|& `$() redirects). Those must use a
// direct ssh command or run without --non-interactive.
func ensureSSHBatchMode(sshCmd string) (string, error) {
	trimmed := strings.TrimSpace(sshCmd)
	if trimmed == "" {
		return "ssh -o BatchMode=yes", nil
	}
	first, rest, ok := splitFirstShellToken(trimmed)
	if !ok || hasUnsupportedShellSyntax(first) || !executableIsSSH(first) || hasUnsupportedShellSyntax(rest) {
		return "", clierrors.ErrorUnsupportedGITSSHCommand
	}
	return first + " -o BatchMode=yes" + rest, nil
}

func executableIsSSH(tok string) bool {
	val, ok := shellTokenValue(tok)
	if !ok || val == "" {
		return false
	}
	return filepath.Base(val) == "ssh"
}

func splitFirstShellToken(s string) (first, rest string, ok bool) {
	quote := rune(0)
	escaped := false
	for i, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if quote == '\'' {
			if r == '\'' {
				quote = 0
			}
			continue
		}
		if quote == '"' {
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				quote = 0
			}
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case '\'', '"':
			quote = r
		case ' ', '\t':
			return s[:i], s[i:], true
		}
	}
	if quote != 0 || escaped {
		return "", "", false
	}
	return s, "", true
}

func shellTokenValue(tok string) (string, bool) {
	var b strings.Builder
	quote := rune(0)
	escaped := false
	for _, r := range tok {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if quote == '\'' {
			if r == '\'' {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if quote == '"' {
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case '\'', '"':
			quote = r
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 || escaped {
		return "", false
	}
	return b.String(), true
}

func hasUnsupportedShellSyntax(s string) bool {
	quote := rune(0)
	escaped := false
	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if quote == '\'' {
			if r == '\'' {
				quote = 0
			}
			continue
		}
		if quote == '"' {
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				quote = 0
				continue
			}
			if r == '$' || r == '`' {
				return true
			}
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case '\'', '"':
			quote = r
		case ';', '|', '&', '`', '$', '(', ')', '<', '>', '\n', '\r':
			return true
		}
	}
	return quote != 0 || escaped
}

// ConfigureRemoteCommand applies non-interactive git environment to a command.
func ConfigureRemoteCommand(cmd *exec.Cmd) error {
	return applyNonInteractiveGit(cmd)
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
	if err := applyNonInteractiveGit(cmd); err != nil {
		return err
	}
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
	if err := applyNonInteractiveGit(cmd); err != nil {
		return err
	}
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
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Commit commits staged changes with the given message
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PushToMain pushes commits to the remote repository on the current branch.
func PushToMain() error {
	cmd := exec.Command("git", "push")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := applyNonInteractiveGit(cmd); err != nil {
		return err
	}
	return cmd.Run()
}

// Pull pulls the latest changes from the remote repository
func Pull(repoDir string) error {
	cmd := exec.Command("git", "pull")
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	if err := applyNonInteractiveGit(cmd); err != nil {
		return err
	}
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
	if err := applyNonInteractiveGit(fetchCmd); err != nil {
		return false, 0, err
	}
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

// RemoteDefaultBranch returns the default branch name advertised by origin.
func RemoteDefaultBranch() (string, error) {
	cmd := exec.Command("git", "ls-remote", "--symref", "origin", "HEAD")
	if err := applyNonInteractiveGit(cmd); err != nil {
		return "", err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("could not determine the remote default branch from origin: %s", msg)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ref:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ref := fields[1]
		if !strings.HasPrefix(ref, "refs/heads/") {
			return "", fmt.Errorf("could not determine the remote default branch from origin")
		}
		name := strings.TrimPrefix(ref, "refs/heads/")
		if name == "" {
			return "", fmt.Errorf("could not determine the remote default branch from origin")
		}
		return name, nil
	}
	return "", fmt.Errorf("could not determine the remote default branch from origin")
}

// CurrentBranch returns the short name of the current branch.
// Detached HEAD is an error.
func CurrentBranch() (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--quiet", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("HEAD is detached")
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", fmt.Errorf("HEAD is detached")
	}
	return name, nil
}

// HeadSHA returns the full commit SHA of HEAD.
func HeadSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RequireDefaultBranch returns the remote default branch when HEAD is on it.
func RequireDefaultBranch() (string, error) {
	defaultBranch, err := RemoteDefaultBranch()
	if err != nil {
		return "", err
	}
	current, err := CurrentBranch()
	if err != nil {
		return "", fmt.Errorf("%s. Checkout the remote default branch %s before deploying", err.Error(), defaultBranch)
	}
	if current != defaultBranch {
		return "", fmt.Errorf("deploy requires the remote default branch %s; current branch is %s. Checkout the default branch; the CLI will not switch branches or force-push", defaultBranch, current)
	}
	return defaultBranch, nil
}

// PushBranch pushes HEAD to origin/<branch> with a fast-forward-only ordinary push.
func PushBranch(branch string) error {
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("could not determine the remote default branch from origin")
	}
	cmd := exec.Command("git", "push", "origin", "HEAD:"+branch)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := applyNonInteractiveGit(cmd); err != nil {
		return err
	}
	return cmd.Run()
}

// AddAll stages every change in the repository, whatever the working directory.
func AddAll() error {
	cmd := exec.Command("git", "add", "-A")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// PushHead pushes HEAD to origin/<branch> without force. A rejected push
// returns git's own output.
func PushHead(branch string) error {
	cmd := exec.Command("git", "push", "origin", "HEAD:"+branch)
	if err := applyNonInteractiveGit(cmd); err != nil {
		return err
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed:\n%s", strings.TrimSpace(string(output)))
	}
	return nil
}

// RemoteHeadState compares HEAD with origin/<branch> after a fetch: "same",
// "behind" (origin has commits HEAD lacks, HEAD is on origin), or "unpushed"
// (HEAD is not on origin).
func RemoteHeadState(branch string) (string, error) {
	fetch := exec.Command("git", "fetch", "--quiet", "origin", branch)
	if err := applyNonInteractiveGit(fetch); err != nil {
		return "", err
	}
	if output, err := fetch.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch failed: %s", strings.TrimSpace(string(output)))
	}
	head, err := HeadSHA()
	if err != nil {
		return "", err
	}
	remote, err := exec.Command("git", "rev-parse", "origin/"+branch).Output()
	if err != nil {
		return "", fmt.Errorf("could not read origin/%s", branch)
	}
	if strings.TrimSpace(string(remote)) == head {
		return "same", nil
	}
	if err := exec.Command("git", "merge-base", "--is-ancestor", "HEAD", "origin/"+branch).Run(); err == nil {
		return "behind", nil
	}
	return "unpushed", nil
}

// PullMerge runs `git pull --no-rebase` in repoDir, so diverged branches merge
// (leaving conflict markers on a conflict) whatever the user's pull config.
// A failure returns git's output.
func PullMerge(repoDir string) error {
	cmd := exec.Command("git", "pull", "--no-rebase")
	cmd.Dir = repoDir
	if err := applyNonInteractiveGit(cmd); err != nil {
		return err
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return clierrors.WrapError("git pull failed: "+string(output), err)
	}
	return nil
}
