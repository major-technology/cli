package utils

import (
	"encoding/json"
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

func TestGetApplicationInfoUsesAppIDEndpointFromChildDir(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "nested", "child")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Write(dir, appConfig()); err != nil {
		t.Fatal(err)
	}

	var seenFromRepo bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != infoPath {
			if r.URL.Path == "/application/from-repo" {
				seenFromRepo = true
			}
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-injected-token" {
			t.Errorf("Authorization = %q, want Bearer test-injected-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, infoBody)
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	info, err := GetApplicationInfo(child)
	if err != nil {
		t.Fatalf("GetApplicationInfo() = %v", err)
	}
	if seenFromRepo {
		t.Fatal("configured app workspace must not call /application/from-repo")
	}
	assertAppInfo(t, info)
}

func TestGetApplicationInfoRejectsSkillTargetWithoutHTTP(t *testing.T) {
	dir := gitRepoWithOrigin(t)
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: testOrgID,
		Target:         workspace.Target{Kind: "skill", SkillID: testAppID},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("skill target must not make HTTP requests, got %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	_, err := GetApplicationInfo(dir)
	if err == nil {
		t.Fatal("expected error for skill target")
	}
}

func TestGetApplicationInfoMalformedConfigDoesNotGitFallback(t *testing.T) {
	dir := gitRepoWithOrigin(t)
	writeRawConfig(t, dir, `{"organizationId":"not-a-uuid"}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("malformed config must not make HTTP requests, got %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	_, err := GetApplicationInfo(dir)
	if err == nil {
		t.Fatal("expected error for malformed config")
	}
	if errors.Is(err, workspace.ErrNotFound) {
		t.Fatal("malformed config must not look like a missing config")
	}
	if !strings.Contains(err.Error(), filepath.Join(dir, ".major", "config.json")) {
		t.Fatalf("error %q should include config path", err)
	}
}

func TestGetApplicationInfoMismatchedOrganizationDoesNotGitFallback(t *testing.T) {
	dir := gitRepoWithOrigin(t)
	if err := workspace.Write(dir, appConfig()); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/application/from-repo" {
			t.Errorf("mismatched organization must not fall back to from-repo")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"applicationId":"`+testAppID+`","organizationId":"`+testOrgID+`","urlSlug":"prototype"}`)
			return
		}
		if r.URL.Path != infoPath {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"applicationId":"`+testAppID+`","organizationId":"33333333-3333-4333-8333-333333333333","urlSlug":"other"}`)
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	_, err := GetApplicationInfo(dir)
	if err == nil {
		t.Fatal("expected organization mismatch error")
	}
}

func TestGetApplicationInfoMissingConfigGitFallbackPersistsWorkspace(t *testing.T) {
	dir := gitRepoWithOrigin(t)
	child := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/application/from-repo" {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"applicationId":"`+testAppID+`","organizationId":"`+testOrgID+`","urlSlug":"prototype"}`)
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	info, err := GetApplicationInfo(child)
	if err != nil {
		t.Fatalf("GetApplicationInfo() = %v", err)
	}
	assertAppInfo(t, info)

	got, err := workspace.Load(child)
	if err != nil {
		t.Fatalf("Load after fallback = %v", err)
	}
	if got.OrganizationID != testOrgID || got.Target.Kind != "app" || got.Target.ApplicationID != testAppID {
		t.Fatalf("persisted config = %+v", got)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".major", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "prototype") || strings.Contains(string(raw), "token") {
		t.Fatalf("persisted config must not include slug or credentials: %s", raw)
	}
	if gitCheckIgnore(t, dir, ".major/config.json") != 0 {
		t.Fatal("legacy lookup must locally ignore .major/config.json")
	}
}

func TestGetApplicationInfoWriteFailureDoesNotClaimSetupSucceeded(t *testing.T) {
	dir := gitRepoWithOrigin(t)
	if err := os.WriteFile(filepath.Join(dir, ".major"), []byte("not-a-directory"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/application/from-repo" {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"applicationId":"`+testAppID+`","organizationId":"`+testOrgID+`","urlSlug":"prototype"}`)
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	_, err := GetApplicationInfo(dir)
	if err == nil {
		t.Fatal("expected write failure after legacy lookup")
	}
	configPath := filepath.Join(dir, ".major", "config.json")
	if !strings.Contains(err.Error(), configPath) {
		t.Fatalf("error %q should include path %q", err, configPath)
	}
}

func TestGetApplicationInfoDeniedInfoDoesNotGitFallback(t *testing.T) {
	dir := gitRepoWithOrigin(t)
	if err := workspace.Write(dir, appConfig()); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/application/from-repo" {
			t.Errorf("scoped info denial must not fall back to from-repo")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"applicationId":"`+testAppID+`","organizationId":"`+testOrgID+`","urlSlug":"prototype"}`)
			return
		}
		if r.URL.Path != infoPath {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(api.ErrorResponse{
			Error: &api.AppErrorDetail{
				InternalCode: 4030,
				ErrorString:  "forbidden",
				StatusCode:   http.StatusForbidden,
			},
		})
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	_, err := GetApplicationInfo(dir)
	if err == nil {
		t.Fatal("expected error after scoped endpoint denial")
	}
}

func TestGetApplicationInfoSiblingMetadataDoesNotWidenOrGitFallback(t *testing.T) {
	const siblingAppID = "44444444-4444-4444-8444-444444444444"
	dir := gitRepoWithOrigin(t)
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: testOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: siblingAppID},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/application/from-repo" {
			t.Errorf("sibling metadata must not fall back to from-repo")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"applicationId":"`+testAppID+`","organizationId":"`+testOrgID+`","urlSlug":"prototype"}`)
			return
		}
		if r.URL.Path != "/applications/"+siblingAppID+"/info" {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(api.ErrorResponse{
			Error: &api.AppErrorDetail{
				InternalCode: 4030,
				ErrorString:  "forbidden",
				StatusCode:   http.StatusForbidden,
			},
		})
	}))
	defer server.Close()
	restoreAPIClient(t, server.URL)

	_, err := GetApplicationInfo(dir)
	if err == nil {
		t.Fatal("expected error when metadata points at sibling app B")
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

func appConfig() workspace.Config {
	return workspace.Config{
		OrganizationID: testOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: testAppID},
	}
}

func assertAppInfo(t *testing.T, info *api.GetApplicationByRepoResponse) {
	t.Helper()
	if info == nil {
		t.Fatal("info is nil")
	}
	if info.ApplicationID != testAppID {
		t.Fatalf("applicationId = %q, want %q", info.ApplicationID, testAppID)
	}
	if info.OrganizationID != testOrgID {
		t.Fatalf("organizationId = %q, want %q", info.OrganizationID, testOrgID)
	}
	if info.URLSlug == nil || *info.URLSlug != "prototype" {
		t.Fatalf("urlSlug = %v, want prototype", info.URLSlug)
	}
}

func gitRepoWithOrigin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "remote", "add", "origin", "git@github.com:acme/prototype.git")
	return dir
}

func writeRawConfig(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, ".major")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "config.json"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
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
