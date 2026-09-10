package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteDefaultBranchFromCloneMetadata(t *testing.T) {
	work, _ := gitCloneFixture(t)
	t.Chdir(work)

	got, err := RemoteDefaultBranch()
	if err != nil {
		t.Fatalf("RemoteDefaultBranch() = %v", err)
	}
	if got != "main" {
		t.Fatalf("RemoteDefaultBranch() = %q, want main", got)
	}
}

func TestCurrentBranchOnDefault(t *testing.T) {
	work, _ := gitCloneFixture(t)
	t.Chdir(work)

	got, err := CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() = %v", err)
	}
	if got != "main" {
		t.Fatalf("CurrentBranch() = %q, want main", got)
	}
}

func TestRequireDefaultBranchDetachedAndFeature(t *testing.T) {
	work, _ := gitCloneFixture(t)
	t.Chdir(work)

	if _, err := RequireDefaultBranch(); err != nil {
		t.Fatalf("on default branch: %v", err)
	}

	runGitDir(t, work, "checkout", "--detach")
	if _, err := RequireDefaultBranch(); err == nil {
		t.Fatal("detached HEAD must fail RequireDefaultBranch")
	}
	runGitDir(t, work, "checkout", "main")
	runGitDir(t, work, "checkout", "-b", "feature")
	if _, err := RequireDefaultBranch(); err == nil {
		t.Fatal("feature branch must fail RequireDefaultBranch")
	}
	if branch := gitOut(t, work, "branch", "--show-current"); branch != "feature" {
		t.Fatalf("must not switch branches, now %q", branch)
	}
}

func TestRemoteDefaultBranchMissingOriginFails(t *testing.T) {
	dir := t.TempDir()
	runGitDir(t, dir, "init", "-b", "main")
	t.Chdir(dir)
	if _, err := RemoteDefaultBranch(); err == nil {
		t.Fatal("missing origin must fail safely")
	}
}

func TestPushBranchUpdatesRemoteWithoutForce(t *testing.T) {
	work, remote := gitCloneFixture(t)
	t.Chdir(work)
	runGitDir(t, work, "commit", "--allow-empty", "-m", "ahead")
	local := gitOut(t, work, "rev-parse", "HEAD")

	if err := PushBranch("main"); err != nil {
		t.Fatalf("PushBranch(main) = %v", err)
	}
	if gitOut(t, remote, "rev-parse", "HEAD") != local {
		t.Fatal("ordinary push must update remote HEAD")
	}

	runGitDir(t, work, "commit", "--allow-empty", "-m", "local diverge")
	other := filepath.Join(t.TempDir(), "other")
	runGitDir(t, filepath.Dir(other), "clone", remote, other)
	runGitDir(t, other, "config", "user.email", "deployer@example.com")
	runGitDir(t, other, "config", "user.name", "Deployer")
	runGitDir(t, other, "commit", "--allow-empty", "-m", "remote diverge")
	runGitDir(t, other, "push", "origin", "HEAD:main")

	if err := PushBranch("main"); err == nil {
		t.Fatal("diverged push must fail without --force")
	}
}

func gitCloneFixture(t *testing.T) (work, remote string) {
	t.Helper()
	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	remote = filepath.Join(root, "origin.git")
	work = filepath.Join(root, "work")
	if err := os.MkdirAll(seed, 0755); err != nil {
		t.Fatal(err)
	}
	runGitDir(t, seed, "init", "-b", "main")
	runGitDir(t, seed, "config", "user.email", "deployer@example.com")
	runGitDir(t, seed, "config", "user.name", "Deployer")
	runGitDir(t, seed, "config", "commit.gpgsign", "false")
	runGitDir(t, seed, "commit", "--allow-empty", "-m", "seed")
	runGitDir(t, root, "clone", "--bare", seed, remote)
	runGitDir(t, root, "clone", remote, work)
	runGitDir(t, work, "config", "user.email", "deployer@example.com")
	runGitDir(t, work, "config", "user.name", "Deployer")
	runGitDir(t, work, "config", "commit.gpgsign", "false")
	return work, remote
}

func runGitDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
