package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
)

const (
	testAppID = "11111111-1111-4111-8111-111111111111"
	testOrgID = "22222222-2222-4222-8222-222222222222"
	infoPath  = "/applications/" + testAppID + "/info"
	infoBody  = `{"applicationId":"11111111-1111-4111-8111-111111111111","organizationId":"22222222-2222-4222-8222-222222222222","urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`
)

func TestGetApplicationAndOrgIDFromDirUsesSharedResolver(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: testOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: testAppID},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/application/from-repo" {
			t.Errorf("helper must not call /application/from-repo")
		}
		if r.URL.Path != infoPath {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, infoBody)
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	appID, orgID, slug, err := getApplicationAndOrgIDFromDir(dir)
	if err != nil {
		t.Fatalf("getApplicationAndOrgIDFromDir() = %v", err)
	}
	if appID != testAppID || orgID != testOrgID || slug != "prototype" {
		t.Fatalf("got app=%s org=%s slug=%s", appID, orgID, slug)
	}
}

func TestPersistAppWorkspaceWritesAppTargetWithoutSlug(t *testing.T) {
	dir := gitRepo(t)
	if err := persistAppWorkspace(dir, testOrgID, testAppID); err != nil {
		t.Fatal(err)
	}
	got, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.OrganizationID != testOrgID || got.Target.Kind != "app" || got.Target.ApplicationID != testAppID {
		t.Fatalf("config = %+v", got)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".major", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "urlSlug") || strings.Contains(string(raw), "token") {
		t.Fatalf("config must not persist slug or credentials: %s", raw)
	}
	if gitCheckIgnore(t, dir, ".major/config.json") != 0 {
		t.Fatal("expected local exclude for .major/config.json")
	}
}

func TestGenerateEnvFileUsesSharedResolver(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: testOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: testAppID},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case infoPath:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, infoBody)
		case "/application/env":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"envVars":{"EXAMPLE":"1"}}`)
		case "/application/from-repo":
			t.Errorf("generateEnvFile must not call /application/from-repo")
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	path, envVars, err := generateEnvFile(dir)
	if err != nil {
		t.Fatalf("generateEnvFile() = %v", err)
	}
	if path != filepath.Join(dir, ".env") {
		t.Fatalf("path = %q", path)
	}
	if envVars["EXAMPLE"] != "1" {
		t.Fatalf("envVars = %#v", envVars)
	}
}

func restoreAPIClient(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(baseURL))
	t.Cleanup(func() {
		singletons.SetAPIClient(prev)
	})
}

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitCheckIgnore(t *testing.T, dir, path string) int {
	t.Helper()
	cmd := exec.Command("git", "check-ignore", "-q", path)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	t.Fatalf("git check-ignore %s: %v", path, err)
	return -1
}
