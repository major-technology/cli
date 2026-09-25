package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/major-technology/cli/clients/config"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

const deployVersionID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

func TestDeployExposesJSONFlag(t *testing.T) {
	if deployCmd.Flags().Lookup("json") == nil {
		t.Fatal("app deploy missing --json")
	}
}

func TestDeployDirtyMissingMessageOrSlugDoesNotCommitOrPush(t *testing.T) {
	t.Run("missing message", func(t *testing.T) {
		assertCloneDeployRefused(t, cloneDeployRefusal{
			infoSlug: "prototype",
			message:  "",
			slug:     "",
			wantErr:  "--message",
			failText: "dirty deploy without --message must fail",
		})
	})
	t.Run("missing slug", func(t *testing.T) {
		assertCloneDeployRefused(t, cloneDeployRefusal{
			infoSlug: "",
			message:  "ship it",
			slug:     "",
			wantErr:  "--slug",
			failText: "first deploy without --slug must fail before commit or push",
		})
	})
}

func TestDeployDirtyDefaultBranchCommitsPushesThenCreatesVersion(t *testing.T) {
	work, remote := cloneFixture(t)
	if err := os.WriteFile(filepath.Join(work, "feature.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	beforeHead := gitOutput(t, work, "rev-parse", "HEAD")

	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "ship dirty", "", true, false)

	cmd, _, _ := deployCaptureCmd(t)
	if err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) }); err != nil {
		t.Fatalf("deploy dirty default branch: %v", err)
	}

	afterHead := gitOutput(t, work, "rev-parse", "HEAD")
	if afterHead == beforeHead {
		t.Fatal("expected a new commit for dirty default-branch deploy")
	}
	if gitOutput(t, work, "status", "--porcelain") != "" {
		t.Fatal("worktree should be clean after deploy commit")
	}
	author := gitOutput(t, work, "log", "-1", "--format=%an <%ae>")
	if author != "Deployer <deployer@example.com>" {
		t.Fatalf("commit author rewritten: %q", author)
	}
	if probe.posts.Load() != 1 {
		t.Fatalf("version creates = %d, want 1", probe.posts.Load())
	}
	if probe.remoteSHA() != afterHead {
		t.Fatalf("remote HEAD at version create = %s, want local %s", probe.remoteSHA(), afterHead)
	}
	if gitOutput(t, remote, "rev-parse", "HEAD") != afterHead {
		t.Fatal("remote default branch was not updated before version create")
	}
}

func TestDeployCleanUnpushedCommitPushesBeforeCreateVersion(t *testing.T) {
	work, remote := cloneFixture(t)
	runGit(t, work, "commit", "--allow-empty", "-m", "local only")
	local := gitOutput(t, work, "rev-parse", "HEAD")
	remoteBefore := gitOutput(t, remote, "rev-parse", "HEAD")
	if local == remoteBefore {
		t.Fatal("precondition: local HEAD must be ahead of remote")
	}

	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", true, false)

	cmd, _, _ := deployCaptureCmd(t)
	if err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) }); err != nil {
		t.Fatalf("deploy clean unpushed: %v", err)
	}

	if gitOutput(t, work, "rev-parse", "HEAD") != local {
		t.Fatal("clean unpushed deploy must not create a new commit")
	}
	if probe.posts.Load() != 1 {
		t.Fatalf("version creates = %d, want 1", probe.posts.Load())
	}
	if probe.remoteSHA() != local {
		t.Fatalf("remote HEAD at version create = %s, want unpushed local %s (pushed before create)", probe.remoteSHA(), local)
	}
}

func TestDeployCleanSynchronizedDefaultBranchDoesNotCreateCommit(t *testing.T) {
	work, remote := cloneFixture(t)
	head := gitOutput(t, work, "rev-parse", "HEAD")

	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", true, false)

	cmd, _, _ := deployCaptureCmd(t)
	if err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) }); err != nil {
		t.Fatalf("deploy synchronized: %v", err)
	}

	if gitOutput(t, work, "rev-parse", "HEAD") != head {
		t.Fatal("synchronized deploy must not create a new commit")
	}
	if probe.posts.Load() != 1 {
		t.Fatalf("version creates = %d, want 1", probe.posts.Load())
	}
	if probe.remoteSHA() != head {
		t.Fatalf("remote HEAD at version create = %s, want %s", probe.remoteSHA(), head)
	}
}

func TestDeployNonDefaultBranchFailsWithoutSwitchOrForce(t *testing.T) {
	work, remote := cloneFixture(t)
	runGit(t, work, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(work, "side.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	before := snapshotGit(t, work)
	remoteBefore := gitOutput(t, remote, "rev-parse", "HEAD")

	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "do not commit", "", true, false)

	cmd, _, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal("non-default branch must fail")
	}

	if gitOutput(t, work, "branch", "--show-current") != "feature" {
		t.Fatalf("must not switch branches, now on %q", gitOutput(t, work, "branch", "--show-current"))
	}
	after := snapshotGit(t, work)
	if after.head != before.head || after.status != before.status || after.refs != before.refs {
		t.Fatalf("git mutated on refused non-default deploy\nbefore=%+v\nafter=%+v", before, after)
	}
	if gitOutput(t, remote, "rev-parse", "HEAD") != remoteBefore {
		t.Fatal("remote default branch changed (possible force push)")
	}
	if probe.posts.Load() != 0 {
		t.Fatalf("version creates = %d, want 0", probe.posts.Load())
	}
}

func TestDeployDetachedHeadFailsWithoutSwitchOrForce(t *testing.T) {
	work, remote := cloneFixture(t)
	runGit(t, work, "checkout", "--detach")
	before := snapshotGit(t, work)
	remoteBefore := gitOutput(t, remote, "rev-parse", "HEAD")

	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", true, false)

	cmd, _, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal("detached HEAD must fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "detach") && !strings.Contains(err.Error(), "default branch") {
		t.Fatalf("error should explain detached/non-default HEAD, got %v", err)
	}
	after := snapshotGit(t, work)
	if after.head != before.head || after.refs != before.refs {
		t.Fatalf("git mutated on detached deploy\nbefore=%+v\nafter=%+v", before, after)
	}
	if gitOutput(t, remote, "rev-parse", "HEAD") != remoteBefore {
		t.Fatal("remote default branch changed")
	}
	if probe.posts.Load() != 0 {
		t.Fatalf("version creates = %d, want 0", probe.posts.Load())
	}
}

func TestDeployRejectedPushDoesNotCreateVersion(t *testing.T) {
	work, remote := cloneFixture(t)
	runGit(t, work, "commit", "--allow-empty", "-m", "will be rejected")
	installRejectHook(t, remote)
	local := gitOutput(t, work, "rev-parse", "HEAD")
	remoteBefore := gitOutput(t, remote, "rev-parse", "HEAD")

	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", true, false)

	cmd, _, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal("rejected push must fail")
	}
	if probe.posts.Load() != 0 {
		t.Fatalf("version creates = %d, want 0 after rejected push", probe.posts.Load())
	}
	if gitOutput(t, work, "rev-parse", "HEAD") != local {
		t.Fatal("rejected push must not rewrite local HEAD")
	}
	if gitOutput(t, remote, "rev-parse", "HEAD") != remoteBefore {
		t.Fatal("rejected push must not update remote HEAD")
	}
}

func TestDeployVersionHashMismatchIsError(t *testing.T) {
	work, remote := cloneFixture(t)
	head := gitOutput(t, work, "rev-parse", "HEAD")
	probe := newDeployProbe(t, remote, "prototype", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", true, true)

	cmd, stdout, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal("mismatched versionHash must be an error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "different commit") {
		t.Fatalf("error must explain a different commit was selected, got %v", err)
	}
	if !strings.Contains(msg, "major app deploy-status --non-interactive --version-id "+deployVersionID) {
		t.Fatalf("known version ID must name deploy-status command, got %v", err)
	}
	if probe.posts.Load() != 1 {
		t.Fatalf("version creates = %d, want 1", probe.posts.Load())
	}
	if probe.remoteSHA() != head {
		t.Fatalf("remote HEAD at version create = %s, want %s", probe.remoteSHA(), head)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("failure must not emit success JSON, stdout=%q", stdout.String())
	}
}

func TestDeployNoWaitJSONSeparatesHashAndDiagnostics(t *testing.T) {
	work, remote := cloneFixture(t)
	head := gitOutput(t, work, "rev-parse", "HEAD")
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", true, true)

	cmd, stdout, stderr := deployCaptureCmd(t)
	if err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) }); err != nil {
		t.Fatalf("deploy --json: %v", err)
	}
	if probe.posts.Load() != 1 {
		t.Fatalf("version creates = %d, want 1", probe.posts.Load())
	}
	got := decodeDeployJSON(t, stdout.String())
	if got["versionId"] != deployVersionID {
		t.Fatalf("versionId = %#v", got["versionId"])
	}
	if got["versionHash"] != head {
		t.Fatalf("versionHash = %#v, want full SHA %s", got["versionHash"], head)
	}
	if got["status"] != "started" {
		t.Fatalf("status = %#v, want started", got["status"])
	}
	if _, ok := got["appUrl"]; ok {
		t.Fatalf("no-wait JSON must not claim a live URL, got %#v", got)
	}
	hint := stderr.String()
	if !strings.Contains(hint, "major app deploy-status --non-interactive --version-id "+deployVersionID) {
		t.Fatalf("stderr must contain deploy-status command, got %q", hint)
	}
}

func TestDeployWaitJSONEmitsFinalStatusOnce(t *testing.T) {
	work, remote := cloneFixture(t)
	head := gitOutput(t, work, "rev-parse", "HEAD")
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", false, true)

	cmd, stdout, stderr := deployCaptureCmd(t)
	if err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) }); err != nil {
		t.Fatalf("deploy wait --json: %v", err)
	}
	if probe.posts.Load() != 1 {
		t.Fatalf("version creates = %d, want 1", probe.posts.Load())
	}
	got := decodeDeployJSON(t, stdout.String())
	if got["versionId"] != deployVersionID || got["versionHash"] != head {
		t.Fatalf("result = %#v", got)
	}
	if got["status"] != "DEPLOYED" {
		t.Fatalf("wait JSON status = %#v, want DEPLOYED not started", got["status"])
	}
	if strings.Contains(stdout.String(), "Status:") {
		t.Fatalf("progress leaked onto stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Status:") && !strings.Contains(stderr.String(), "Deployed") {
		t.Fatalf("wait progress should go to stderr, got %q", stderr.String())
	}
}

func TestDeployWaitFailedJSONExitsNonzero(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	probe.waitStatus = "BUILD_FAILED"
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", false, true)

	cmd, stdout, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal("failed terminal status must exit nonzero")
	}
	if !strings.Contains(err.Error(), "BUILD_FAILED") {
		t.Fatalf("error must name failed status, got %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("failed wait must not emit success JSON, stdout=%q", stdout.String())
	}
}

func TestDeployWaitFailedJSONCarriesDeploymentError(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	probe.waitStatus = "BUILD_FAILED"
	probe.deploymentError = "query extraction timed out"
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", false, true)

	cmd, stdout, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil || err.Error() != "deployment failed with status BUILD_FAILED: query extraction timed out" {
		t.Fatalf("--json failure must carry deploymentError, got %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("failed wait must not emit success JSON, stdout=%q", stdout.String())
	}
}

func TestPromptForDeployURLJSONWritesBannersToStderr(t *testing.T) {
	flagDeployJSON = true
	t.Cleanup(func() { flagDeployJSON = false })

	prev := collectFirstDeploySlug
	collectFirstDeploySlug = func(cmd *cobra.Command, slug *string) error {
		*slug = "my-app"
		return nil
	}
	t.Cleanup(func() { collectFirstDeploySlug = prev })

	prevCfg := singletons.GetConfig()
	singletons.SetConfig(&config.Config{AppURLSuffix: "example.test"})
	t.Cleanup(func() { singletons.SetConfig(prevCfg) })

	cmd, stdout, stderr := deployCaptureCmd(t)
	got, err := promptForDeployURL(cmd)
	if err != nil {
		t.Fatalf("promptForDeployURL: %v", err)
	}
	if got != "my-app" {
		t.Fatalf("slug = %q, want my-app", got)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("JSON first-deploy prompt must not write to stdout, got %q", stdout.String())
	}
	out := stderr.String()
	if !strings.Contains(out, "First deploy") {
		t.Fatalf("stderr must contain first-deploy banner, got %q", out)
	}
	if !strings.Contains(out, "https://my-app.example.test") {
		t.Fatalf("stderr must contain selected URL, got %q", out)
	}
}

func TestDeployWaitStatusConnectionLostIncludesDeployStatusCommand(t *testing.T) {
	work, remote := cloneFixture(t)
	probe := newDeployProbe(t, remote, "prototype", "", http.StatusOK)
	probe.dropStatus = true
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", false, true)

	cmd, stdout, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal("status-endpoint connection loss after create must fail")
	}
	if probe.posts.Load() != 1 {
		t.Fatalf("version creates = %d, want 1 (no retry)", probe.posts.Load())
	}
	msg := err.Error()
	if !strings.Contains(msg, "major app deploy-status --non-interactive --version-id "+deployVersionID) {
		t.Fatalf("known version ID must name deploy-status command, got %v", err)
	}
	if strings.Contains(msg, "major app info --non-interactive --json") {
		t.Fatalf("must not use info discovery when version ID is known, got %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("failure must not emit success JSON, stdout=%q", stdout.String())
	}
}

func TestDeployConnectionLostDoesNotRetry(t *testing.T) {
	work, remote := cloneFixture(t)
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/applications/"+niAppID+"/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID)
		case r.URL.Path == "/applications/versions":
			posts.Add(1)
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Error("server cannot hijack")
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = remote
			conn.Close()
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	restoreAPIClient(t, srv.URL)
	t.Chdir(work)
	setDeployCommandFlags(t, "", "", true, true)

	cmd, stdout, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal("accepted-but-connection-lost must fail")
	}
	if posts.Load() != 1 {
		t.Fatalf("version creates = %d, want exactly 1 (no retry)", posts.Load())
	}
	if !strings.Contains(err.Error(), "major app info --non-interactive --json") {
		t.Fatalf("unknown result must name info discovery command, got %v", err)
	}
	if strings.Contains(err.Error(), "major app deploy-status --non-interactive --version-id") {
		t.Fatalf("must not invent a version ID after connection loss, got %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("failure must not emit success JSON, stdout=%q", stdout.String())
	}
}

type cloneDeployRefusal struct {
	infoSlug string
	message  string
	slug     string
	wantErr  string
	failText string
}

func assertCloneDeployRefused(t *testing.T, spec cloneDeployRefusal) {
	t.Helper()
	work, remote := cloneFixture(t)
	if err := os.WriteFile(filepath.Join(work, "dirty.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	before := snapshotGit(t, work)
	probe := newDeployProbe(t, remote, spec.infoSlug, "", http.StatusOK)
	t.Chdir(work)
	setDeployCommandFlags(t, spec.message, spec.slug, true, false)

	cmd, _, _ := deployCaptureCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal(spec.failText)
	}
	if !strings.Contains(err.Error(), spec.wantErr) {
		t.Fatalf("error must name %s, got %v", spec.wantErr, err)
	}
	if probe.posts.Load() != 0 {
		t.Fatalf("version creates = %d, want 0", probe.posts.Load())
	}
	after := snapshotGit(t, work)
	if after != before {
		t.Fatalf("git state changed after refused deploy\nbefore=%+v\nafter=%+v", before, after)
	}
}

func cloneFixture(t *testing.T) (work, remote string) {
	t.Helper()
	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	remote = filepath.Join(root, "origin.git")
	work = filepath.Join(root, "work")
	if err := os.MkdirAll(seed, 0755); err != nil {
		t.Fatal(err)
	}
	runGit(t, seed, "init", "-b", "main")
	runGit(t, seed, "config", "user.email", "deployer@example.com")
	runGit(t, seed, "config", "user.name", "Deployer")
	runGit(t, seed, "config", "commit.gpgsign", "false")
	runGit(t, seed, "commit", "--allow-empty", "-m", "seed")
	runGit(t, root, "clone", "--bare", seed, remote)
	runGit(t, root, "clone", remote, work)
	runGit(t, work, "config", "user.email", "deployer@example.com")
	runGit(t, work, "config", "user.name", "Deployer")
	runGit(t, work, "config", "commit.gpgsign", "false")
	if err := workspace.Write(work, workspace.Config{
		OrganizationID: niOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: niAppID},
	}); err != nil {
		t.Fatal(err)
	}
	if err := workspace.IgnoreLocalConfig(work); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	return work, remote
}

func setDeployCommandFlags(t *testing.T, message, slug string, noWait, jsonOut bool) {
	t.Helper()
	flagDeployMessage = message
	flagDeploySlug = slug
	flagDeployNoWait = noWait
	flagDeployJSON = jsonOut
	t.Cleanup(func() {
		flagDeployMessage = ""
		flagDeploySlug = ""
		flagDeployNoWait = false
		flagDeployJSON = false
	})
}

func deployCaptureCmd(t *testing.T) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	if err := root.PersistentFlags().Set("non-interactive", "true"); err != nil {
		t.Fatal(err)
	}
	child := &cobra.Command{Use: "deploy"}
	root.AddCommand(child)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	child.SetIn(strings.NewReader(""))
	child.SetOut(stdout)
	child.SetErr(stderr)
	return child, stdout, stderr
}

type deployProbe struct {
	t               *testing.T
	remote          string
	posts           atomic.Int32
	sha             atomic.Value
	hash            string
	status          int
	infoSlug        string
	waitStatus      string
	deploymentError string
	postedSlug      atomic.Value
	dropStatus      bool
}

func newDeployProbe(t *testing.T, remote, infoSlug, versionHash string, status int) *deployProbe {
	t.Helper()
	p := &deployProbe{t: t, remote: remote, hash: versionHash, status: status, infoSlug: infoSlug}
	srv := httptest.NewServer(http.HandlerFunc(p.serve))
	t.Cleanup(srv.Close)
	restoreAPIClient(t, srv.URL)
	return p
}

func (p *deployProbe) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/applications/"+niAppID+"/info":
		w.Header().Set("Content-Type", "application/json")
		slug := "null"
		if p.infoSlug != "" {
			slug = fmt.Sprintf("%q", p.infoSlug)
		}
		fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":%s,"name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID, slug)
	case r.URL.Path == "/applications/versions":
		p.posts.Add(1)
		var body struct {
			AppURL string `json:"appURL"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		p.postedSlug.Store(body.AppURL)
		sha := gitSHA(p.remote)
		p.sha.Store(sha)
		hash := p.hash
		if hash == "" {
			hash = sha
		}
		if p.status != http.StatusOK {
			w.WriteHeader(p.status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"versionId":%q,"versionHash":%q}`, deployVersionID, hash)
	case r.URL.Path == "/applications/versions/status" || strings.Contains(r.URL.Path, "versions/status"):
		if p.dropStatus {
			hj, ok := w.(http.Hijacker)
			if !ok {
				p.t.Error("server cannot hijack")
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				p.t.Errorf("hijack: %v", err)
				return
			}
			conn.Close()
			return
		}
		waitStatus := p.waitStatus
		if waitStatus == "" {
			waitStatus = "DEPLOYED"
		}
		w.Header().Set("Content-Type", "application/json")
		appURL := ""
		if waitStatus == "DEPLOYED" {
			appURL = "https://prototype.example.test"
		}
		fmt.Fprintf(w, `{"status":%q,"deploymentError":%q,"app_url":%q}`, waitStatus, p.deploymentError, appURL)
	default:
		p.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (p *deployProbe) remoteSHA() string {
	v, _ := p.sha.Load().(string)
	return v
}

func gitSHA(dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func installRejectHook(t *testing.T, remote string) {
	t.Helper()
	hook := filepath.Join(remote, "hooks", "pre-receive")
	body := "#!/bin/sh\necho rejected >&2\nexit 1\n"
	if err := os.WriteFile(hook, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
}

func decodeDeployJSON(t *testing.T, stdout string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(stdout))
	var got map[string]any
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("stdout JSON: %v raw=%q", err, stdout)
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err == nil {
		t.Fatalf("stdout contained more than one JSON value: %s", stdout)
	}
	return got
}
