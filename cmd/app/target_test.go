package app

import (
	"bytes"
	stderrors "errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/target"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// appContext is a TargetContext for the app workspace at work, with its own
// output buffers and --json flag.
func appContext(t *testing.T, work, notes string, jsonOut bool) (*target.TargetContext, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfg, err := workspace.Load(work)
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("json", jsonOut, "")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	return &target.TargetContext{Root: work, Config: cfg, API: singletons.GetAPIClient(), Command: cmd, Notes: notes}, stdout, stderr
}

// commitOnSecondClone pushes a commit that edits file from a second clone.
func commitOnSecondClone(t *testing.T, remote, file, content string) {
	t.Helper()
	other := filepath.Join(t.TempDir(), "other")
	runGit(t, filepath.Dir(other), "clone", remote, other)
	runGit(t, other, "config", "user.email", "other@example.com")
	runGit(t, other, "config", "user.name", "Other")
	runGit(t, other, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(other, file), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, other, "add", file)
	runGit(t, other, "commit", "-m", "remote edit")
	runGit(t, other, "push", "origin", "HEAD:main")
}

func TestAppPushWithRemoteAheadFailsWithGitTextAndPullHint(t *testing.T) {
	work, remote := cloneFixture(t)
	commitOnSecondClone(t, remote, "remote.txt", "theirs")
	remoteBefore := gitSHA(remote)
	if err := os.WriteFile(filepath.Join(work, "local.txt"), []byte("mine"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	ctx, stdout, _ := appContext(t, work, "local edit", true)

	_, err := appTarget{}.Push(ctx)
	if err == nil {
		t.Fatal("push over a newer remote must fail")
	}
	if !strings.Contains(err.Error(), "rejected") || !strings.Contains(err.Error(), "run major pull, resolve conflicts, then major push") {
		t.Fatalf("error must carry git's text and the pull hint: %v", err)
	}
	if gitSHA(remote) != remoteBefore {
		t.Fatal("remote moved: the push was forced")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout after an error: %q", stdout.String())
	}
}

func TestAppPushDirtyTreeRequiresMessageThenCommitsAndPushes(t *testing.T) {
	work, remote := cloneFixture(t)
	if err := os.MkdirAll(filepath.Join(work, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "src", "page.tsx"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(work, "src"))
	ctx, _, _ := appContext(t, work, "", false)
	if _, err := (appTarget{}).Push(ctx); err == nil || !strings.Contains(err.Error(), "-m") {
		t.Fatalf("dirty push without -m: %v", err)
	}
	ctx.Notes = "add page"
	result, err := appTarget{}.Push(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gitSHA(remote) != gitSHA(work) || result.(appGitResult).Commit != gitSHA(work) {
		t.Fatalf("push did not land HEAD on origin")
	}
	if gitOutput(t, work, "status", "--porcelain") != "" {
		t.Fatal("tree should be clean after the commit")
	}
}

func TestAppPullConflictLeavesMarkersAndFails(t *testing.T) {
	// Default git config: no global or system pull.rebase/pull.ff reaches git.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	work, remote := cloneFixture(t)
	commitOnSecondClone(t, remote, "shared.txt", "theirs\n")
	if err := os.WriteFile(filepath.Join(work, "shared.txt"), []byte("mine\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "shared.txt")
	runGit(t, work, "commit", "-m", "local edit")
	t.Chdir(work)
	ctx, stdout, _ := appContext(t, work, "", true)

	_, err := appTarget{}.Pull(ctx)
	if err == nil {
		t.Fatal("conflicting pull must fail")
	}
	if !strings.Contains(err.Error(), "CONFLICT") {
		t.Fatalf("error must carry git's conflict text: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(work, "shared.txt"))
	if !strings.Contains(string(data), "<<<<<<<") {
		t.Fatalf("conflict markers missing: %q", data)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout after an error: %q", stdout.String())
	}
}

// installFakePnpm puts a pnpm on PATH that prints a line and exits with code.
func installFakePnpm(t *testing.T, code string) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\necho lint-output\nexit " + code + "\n"
	if err := os.WriteFile(filepath.Join(bin, "pnpm"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestAppValidateReturnsLintExitCode(t *testing.T) {
	work, _ := cloneFixture(t)
	if err := os.WriteFile(filepath.Join(work, "package.json"), []byte(`{"scripts":{"lint":"eslint"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)

	installFakePnpm(t, "3")
	ctx, stdout, stderr := appContext(t, work, "", true)
	_, err := appTarget{}.Validate(ctx)
	var exitErr *clierrors.ExitCodeError
	if !stderrors.As(err, &exitErr) || exitErr.Code != 3 {
		t.Fatalf("want exit code 3, got %v", err)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "lint-output") {
		t.Fatalf("--json lint output must go to stderr: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	installFakePnpm(t, "0")
	ctx, _, _ = appContext(t, work, "", false)
	if result, err := (appTarget{}).Validate(ctx); err != nil || !result.(appValidateResult).Valid {
		t.Fatalf("passing lint: %v %v", result, err)
	}
}

func TestAppValidateWithoutLintScriptFails(t *testing.T) {
	work, _ := cloneFixture(t)
	if err := os.WriteFile(filepath.Join(work, "package.json"), []byte(`{"scripts":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	ctx, _, _ := appContext(t, work, "", false)
	if _, err := (appTarget{}).Validate(ctx); err == nil || !strings.Contains(err.Error(), "no lint script") {
		t.Fatalf("missing lint script: %v", err)
	}
}

func TestAppPublishWithUnpushedHeadFailsBeforeDeploying(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	runGit(t, work, "commit", "--allow-empty", "-m", "local only")
	t.Chdir(work)
	ctx, stdout, _ := appContext(t, work, "", true)

	_, err := appTarget{}.Publish(ctx)
	if err == nil || !strings.Contains(err.Error(), "run major push first") {
		t.Fatalf("unpushed HEAD: %v", err)
	}
	if probe.posts.Load() != 0 {
		t.Fatal("a version was created for an unpushed HEAD")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout after an error: %q", stdout.String())
	}
}

func TestAppPublishDeploysPushedHeadAndWaits(t *testing.T) {
	work, remote := cloneFixture(t)
	newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	ctx, _, stderr := appContext(t, work, "", true)

	var result any
	err := runWithDeadline(t, 8*time.Second, func() error {
		var err error
		result, err = appTarget{}.Publish(ctx)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	published := result.(appPublishResult)
	if published.Status != "DEPLOYED" || published.VersionHash != gitSHA(work) {
		t.Fatalf("result: %+v", published)
	}
	if !strings.Contains(stderr.String(), "Status:") {
		t.Fatalf("progress should go to stderr: %q", stderr.String())
	}
}

func TestAppPublishFailureCarriesDeploymentError(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	probe.waitStatus = "BUILD_FAILED"
	probe.deploymentError = "query extraction timed out"
	t.Chdir(work)
	ctx, stdout, _ := appContext(t, work, "", true)

	err := runWithDeadline(t, 8*time.Second, func() error {
		_, err := appTarget{}.Publish(ctx)
		return err
	})
	if err == nil || err.Error() != "deployment failed with status BUILD_FAILED: query extraction timed out" {
		t.Fatalf("error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout after an error: %q", stdout.String())
	}
}

// runAppAction runs `major <action> --json` through a fresh copy of the shared
// top-level command, so flags never leak between tests.
func runAppAction(t *testing.T, action string, flags map[string]string) (string, error) {
	t.Helper()
	var cmd *cobra.Command
	for _, candidate := range target.TargetCommands() {
		if candidate.Use == action {
			cmd = candidate
		}
	}
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", true, "")
	root.AddCommand(cmd)
	flags["json"] = "true"
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	var err error
	runWithDeadline(t, 8*time.Second, func() error {
		err = cmd.RunE(cmd, nil)
		return nil
	})
	return stdout.String(), err
}

// B6 for apps: one JSON object per successful action, nothing on stdout after an error.
func TestAppActionsFollowTheOutputContract(t *testing.T) {
	work, remote := cloneFixture(t)
	newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	if err := os.WriteFile(filepath.Join(work, "package.json"), []byte(`{"scripts":{"lint":"eslint"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	installFakePnpm(t, "0")
	t.Chdir(work)

	for _, step := range []struct {
		action string
		flags  map[string]string
	}{
		{"push", map[string]string{"message": "add package.json"}},
		{"validate", map[string]string{}},
		{"publish", map[string]string{"yes": "true"}},
		{"pull", map[string]string{}},
	} {
		stdout, err := runAppAction(t, step.action, step.flags)
		if err != nil {
			t.Fatalf("%s: %v", step.action, err)
		}
		decodeDeployJSON(t, stdout)
	}

	installFakePnpm(t, "2")
	runGit(t, work, "commit", "--allow-empty", "-m", "unpushed")
	for _, step := range []struct {
		action string
		flags  map[string]string
	}{
		{"validate", map[string]string{}},
		{"publish", map[string]string{"yes": "true"}},
	} {
		stdout, err := runAppAction(t, step.action, step.flags)
		if err == nil || stdout != "" {
			t.Fatalf("%s: err=%v stdout=%q", step.action, err, stdout)
		}
	}
}

func TestAppPublishFirstDeployUsesSlugFlag(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "", "", http.StatusOK)
	t.Chdir(work)

	stdout, err := runAppAction(t, "publish", map[string]string{"yes": "true", "slug": "my-app"})
	if err != nil {
		t.Fatal(err)
	}
	result := decodeDeployJSON(t, stdout)
	if result["status"] != "DEPLOYED" {
		t.Fatalf("result: %v", result)
	}
	if got, _ := probe.postedSlug.Load().(string); got != "my-app" {
		t.Fatalf("deployed with slug %q, want my-app", got)
	}
}

func TestAppPublishFirstDeployWithoutSlugNamesTheFlag(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "", "", http.StatusOK)
	t.Chdir(work)

	stdout, err := runAppAction(t, "publish", map[string]string{"yes": "true"})
	if err == nil || !strings.Contains(err.Error(), "major publish --slug <slug>") {
		t.Fatalf("error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("stdout after an error: %q", stdout)
	}
	if probe.posts.Load() != 0 {
		t.Fatal("a version was created without a slug")
	}
}

// Mirrors `major app deploy --slug`: an app that has a slug keeps it.
func TestAppPublishIgnoresSlugFlagOnceTheAppHasOne(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)

	stdout, err := runAppAction(t, "publish", map[string]string{"yes": "true", "slug": "other"})
	if err != nil {
		t.Fatal(err)
	}
	decodeDeployJSON(t, stdout)
	if got, _ := probe.postedSlug.Load().(string); got != "prototype" {
		t.Fatalf("deployed with slug %q, want the existing prototype", got)
	}
}
