package app

import (
	"bytes"
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
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

const (
	niAppID = "11111111-1111-4111-8111-111111111111"
	niOrgID = "22222222-2222-4222-8222-222222222222"
)

func TestCreateNonInteractiveRequiresNameAndDescription(t *testing.T) {
	cmd, writes := setupCreateCommand(t)
	flagAppName = ""
	flagAppDescription = ""
	t.Cleanup(func() {
		flagAppName = ""
		flagAppDescription = ""
	})

	err := runWithDeadline(t, 8*time.Second, func() error { return runCreate(cmd) })
	if err == nil {
		t.Fatal("missing name/description must fail")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--name") {
		t.Fatalf("error must name --name, got %v", err)
	}
	if !strings.Contains(msg, "--description") {
		t.Fatalf("error must name --description, got %v", err)
	}
	if writes.Load() != 0 {
		t.Fatalf("CreateApplication calls = %d, want 0", writes.Load())
	}
}

func TestCloneNonInteractiveRequiresAppIDWhenMultiple(t *testing.T) {
	keyring.MockInit()
	if err := mjrToken.StoreDefaultOrg(niOrgID, "Test Org"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.URL.Path != "/organizations/applications" {
			writes.Add(1)
		}
		if r.URL.Path == "/organizations/applications" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"applications":[{"id":"a1","name":"One","githubRepositoryName":"one","cloneUrlSsh":"git@github.com:o/one.git","cloneUrlHttps":"https://github.com/o/one.git"},{"id":"a2","name":"Two","githubRepositoryName":"two","cloneUrlSsh":"git@github.com:o/two.git","cloneUrlHttps":"https://github.com/o/two.git"}]}`)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	restoreAPIClient(t, srv.URL)

	cmd := nonInteractiveCmd(t)
	flagAppID = ""
	t.Cleanup(func() { flagAppID = "" })

	err := runWithDeadline(t, 8*time.Second, func() error { return runClone(cmd) })
	if err == nil {
		t.Fatal("clone without --app-id must fail when multiple apps exist")
	}
	if !strings.Contains(err.Error(), "--app-id") {
		t.Fatalf("error must name --app-id, got %v", err)
	}
	if writes.Load() != 0 {
		t.Fatalf("mutation calls = %d, want 0", writes.Load())
	}
}

func TestDeployNonInteractiveMissingMessageLeavesGitUnchanged(t *testing.T) {
	assertRefusedDeployLeavesGitUnchanged(t, deployGitRefusal{
		infoBody: fmt.Sprintf(`{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID),
		message:  "",
		slug:     "",
		wantErr:  "--message",
		failText: "dirty deploy without --message must fail",
	})
}

func TestDeployNonInteractiveMissingSlugLeavesGitUnchanged(t *testing.T) {
	assertRefusedDeployLeavesGitUnchanged(t, deployGitRefusal{
		infoBody: fmt.Sprintf(`{"applicationId":%q,"organizationId":%q,"urlSlug":null,"name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID),
		message:  "ship it",
		slug:     "",
		wantErr:  "--slug",
		failText: "first deploy without --slug must fail",
	})
}

type deployGitRefusal struct {
	infoBody string
	message  string
	slug     string
	wantErr  string
	failText string
}

func assertRefusedDeployLeavesGitUnchanged(t *testing.T, spec deployGitRefusal) {
	t.Helper()
	dir := preparedAppRepo(t, "")
	runGit(t, dir, "commit", "-m", "initial", "--allow-empty")
	attachBareOrigin(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	before := snapshotGit(t, dir)

	var versionWrites atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/applications/"+niAppID+"/info" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, spec.infoBody)
			return
		}
		if r.URL.Path == "/applications/versions" {
			versionWrites.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	restoreAPIClient(t, srv.URL)

	t.Chdir(dir)
	flagDeployMessage = spec.message
	flagDeploySlug = spec.slug
	t.Cleanup(func() {
		flagDeployMessage = ""
		flagDeploySlug = ""
	})

	cmd := nonInteractiveCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd) })
	if err == nil {
		t.Fatal(spec.failText)
	}
	if !strings.Contains(err.Error(), spec.wantErr) {
		t.Fatalf("error must name %s, got %v", spec.wantErr, err)
	}
	if versionWrites.Load() != 0 {
		t.Fatalf("version create calls = %d, want 0", versionWrites.Load())
	}
	after := snapshotGit(t, dir)
	if after != before {
		t.Fatalf("git state changed after refused deploy\nbefore=%+v\nafter=%+v", before, after)
	}
}

type gitSnapshot struct {
	head   string
	status string
	remote string
	refs   string
}

func attachBareOrigin(t *testing.T, dir string) {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "origin.git")
	if err := os.MkdirAll(remote, 0755); err != nil {
		t.Fatal(err)
	}
	runGit(t, remote, "init", "--bare")
	runGit(t, dir, "remote", "add", "origin", remote)
	runGit(t, dir, "push", "-u", "origin", "HEAD:main")
}

func snapshotGit(t *testing.T, dir string) gitSnapshot {
	t.Helper()
	return gitSnapshot{
		head:   gitOutput(t, dir, "rev-parse", "HEAD"),
		status: gitOutput(t, dir, "status", "--porcelain"),
		remote: gitOutput(t, dir, "remote", "-v"),
		refs:   gitOutput(t, dir, "ls-remote", "origin"),
	}
}

func TestStartNonInteractiveSkipsThemeUpgradeByDefault(t *testing.T) {
	dir := preparedAppRepo(t, "prototype")
	cssPath := filepath.Join(dir, "app", "theme.css")
	if err := os.MkdirAll(filepath.Dir(cssPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cssPath, []byte("/* keep */"), 0644); err != nil {
		t.Fatal(err)
	}

	var upgrades atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/applications/"+niAppID+"/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID)
		case r.URL.Path == "/application/"+niAppID+"/theme-version":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"appThemeVersion":1,"latestThemeVersion":2,"upgradeAvailable":true}`)
		case strings.Contains(r.URL.Path, "upgrade-theme"):
			upgrades.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	restoreAPIClient(t, srv.URL)
	t.Chdir(dir)

	flagUpgradeTheme = false
	t.Cleanup(func() { flagUpgradeTheme = false })

	cmd := nonInteractiveCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return handleThemeSync(cmd) })
	if err != nil {
		t.Fatalf("handleThemeSync() = %v", err)
	}
	if upgrades.Load() != 0 {
		t.Fatalf("UpgradeTheme calls = %d, want 0 without --upgrade-theme", upgrades.Load())
	}
	got, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "/* keep */" {
		t.Fatalf("theme files changed: %q", got)
	}
}

func TestConfigureNonInteractivePrintsURL(t *testing.T) {
	dir := preparedAppRepo(t, "prototype")
	orig := utils.BrowserStart
	opened := []string{}
	utils.BrowserStart = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	t.Cleanup(func() { utils.BrowserStart = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/applications/"+niAppID+"/info" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID)
			return
		}
		t.Errorf("unexpected request: %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	restoreAPIClient(t, srv.URL)
	prevCfg := singletons.GetConfig()
	singletons.SetConfig(&config.Config{FrontendURI: "https://app.example.test"})
	t.Cleanup(func() { singletons.SetConfig(prevCfg) })
	t.Chdir(dir)

	cmd := nonInteractiveCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runConfigure(cmd) })
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if len(opened) != 0 {
		t.Fatalf("opened browser: %v", opened)
	}
	if !strings.Contains(cmd.OutOrStdout().(*bytes.Buffer).String(), "https://app.example.test/home?dialog=app-settings&appId="+niAppID) {
		t.Fatalf("output must print destination, got %q", cmd.OutOrStdout().(*bytes.Buffer).String())
	}
}

func setupCreateCommand(t *testing.T) (*cobra.Command, *atomic.Int32) {
	t.Helper()
	keyring.MockInit()
	if err := mjrToken.StoreDefaultOrg(niOrgID, "Test Org"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/applications" {
			writes.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/themes") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"themes":[]}`)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	restoreAPIClient(t, srv.URL)
	return nonInteractiveCmd(t), &writes
}

func nonInteractiveCmd(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)
	child.SetIn(strings.NewReader(""))
	out := &bytes.Buffer{}
	child.SetOut(out)
	child.SetErr(out)
	return child
}

func preparedAppRepo(t *testing.T, slug string) string {
	t.Helper()
	dir := gitRepo(t)
	runGit(t, dir, "commit", "--allow-empty", "-m", "base")
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: niOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: niAppID},
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	_ = slug
	return dir
}

func gitOutput(t *testing.T, dir string, args ...string) string {
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

func runWithDeadline(t *testing.T, d time.Duration, fn func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		t.Fatalf("command did not complete within %s (possible prompt)", d)
		return nil
	}
}
