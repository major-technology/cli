package project

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCompileResultCompiled(t *testing.T) {
	dir := writeValidProject(t)

	body := buildCompileResult(dir, "p-1", "org-1", "abc123", "1.2.3")

	if body.Status != "compiled" {
		t.Fatalf("status = %q, want compiled", body.Status)
	}
	if body.ProjectID != "p-1" || body.OrganizationID != "org-1" || body.CommitHash != "abc123" {
		t.Fatalf("identity fields wrong: %+v", body)
	}
	if body.CompilerVersion != "1.2.3" {
		t.Fatalf("compilerVersion = %q", body.CompilerVersion)
	}
	if len(body.CompiledConfig) == 0 {
		t.Fatal("compiledConfig empty")
	}
	if body.CompileError != "" {
		t.Fatalf("unexpected compileError: %q", body.CompileError)
	}
}

func TestBuildCompileResultFailed(t *testing.T) {
	dir := t.TempDir()
	// project.json exists but agent is invalid
	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{"name":"T"}`), 0644); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(dir, "src", "agents", "bad")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent.json"), []byte(`{"slug":"bad"}`), 0644); err != nil {
		t.Fatal(err)
	}

	body := buildCompileResult(dir, "p-1", "org-1", "abc123", "1.2.3")

	if body.Status != "failed" {
		t.Fatalf("status = %q, want failed", body.Status)
	}
	if body.CompileError == "" {
		t.Fatal("expected compileError to describe the issues")
	}
	if len(body.CompiledConfig) != 0 {
		t.Fatal("failed compile must not carry a config")
	}
}

func TestPostCompileResult(t *testing.T) {
	var gotAuth string
	var gotBody compileResultRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("x-major-cross-server-jwt")
		if r.URL.Path != "/internal/projects/compile-result" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	body := compileResultRequest{ProjectID: "p-1", OrganizationID: "org-1", CommitHash: "abc", CompilerVersion: "dev", Status: "failed", CompileError: "boom"}

	if err := postCompileResult(srv.URL, "jwt-token", body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "jwt-token" {
		t.Fatalf("cross-server jwt header = %q", gotAuth)
	}
	if gotBody.ProjectID != "p-1" || gotBody.Status != "failed" {
		t.Fatalf("body not delivered: %+v", gotBody)
	}
}

func TestPostCompileResultServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	err := postCompileResult(srv.URL, "jwt", compileResultRequest{})
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

// TestCloneAtCommitRedactsToken locks sanitize() into cloneAtCommit's error
// path: a fetch failure against a bad local path is guaranteed to build its
// error message from the tokenized command line (see cloneAtCommit's
// "label"), so this is a deterministic, offline regression guard against a
// future refactor (e.g. swapping CombinedOutput or moving URL construction)
// silently dropping the sanitize() call and leaking GITHUB_TOKEN.
func TestCloneAtCommitRedactsToken(t *testing.T) {
	const token = "ghs_faketoken_do_not_leak_123"
	dir := filepath.Join(t.TempDir(), "workdir")

	err := cloneAtCommit("file:///nonexistent-dir-xyz-abc/should-not-exist/repo", token, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", dir)
	if err == nil {
		t.Fatal("expected error cloning a nonexistent local repo")
	}

	msg := err.Error()

	if strings.Contains(msg, token) {
		t.Fatalf("error leaked the raw token: %s", msg)
	}
	if strings.Contains(msg, "x-access-token:"+token) {
		t.Fatalf("error leaked the tokenized URL: %s", msg)
	}
	if !strings.Contains(msg, "***") {
		t.Fatalf("expected sanitized placeholder (***) in error: %s", msg)
	}
}

// initGitRepoWithFiles creates a local git repo with the given files
// (relative path -> content), commits them, and returns a file:// URL for
// the repo plus the resulting commit SHA - a real, offline git source for
// exercising cloneAtCommit's single-commit fetch-by-SHA end to end.
func initGitRepoWithFiles(t *testing.T, files map[string]string) (repoURL, sha string) {
	t.Helper()
	dir := t.TempDir()

	for relPath, content := range files {
		full := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init", "--quiet")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("add", ".")
	run("commit", "--quiet", "-m", "init")

	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}

	return "file://" + dir, strings.TrimSpace(string(out))
}

// TestCompileAndReportRunEExitSemantics locks in the central exit contract:
// a delivered report - compiled OR failed - is a successful run (nil, exit
// 0); only infrastructure failures (missing env, unreachable report
// endpoint) are errors (non-nil, nonzero exit).
func TestCompileAndReportRunEExitSemantics(t *testing.T) {
	validRepoURL, validSHA := initGitRepoWithFiles(t, map[string]string{
		"project.json":            `{"name":"T"}`,
		"src/agents/a/agent.json": `{"slug":"a","name":"A","systemPrompt":"hi"}`,
	})
	brokenRepoURL, brokenSHA := initGitRepoWithFiles(t, map[string]string{
		"project.json":              `{"name":"T"}`,
		"src/agents/bad/agent.json": `{"slug":"bad"}`,
	})

	tests := []struct {
		name               string
		repoURL            string
		commitHash         string
		unsetEnv           string
		serverStatus       int
		wantErr            bool
		wantReportedStatus string // "" = no report expected
	}{
		{
			name:               "valid project compiles, reports compiled, exits clean",
			repoURL:            validRepoURL,
			commitHash:         validSHA,
			serverStatus:       http.StatusOK,
			wantErr:            false,
			wantReportedStatus: "compiled",
		},
		{
			name:               "broken project reports failed and still exits clean",
			repoURL:            brokenRepoURL,
			commitHash:         brokenSHA,
			serverStatus:       http.StatusOK,
			wantErr:            false,
			wantReportedStatus: "failed",
		},
		{
			name:         "missing env var is an infrastructure failure",
			repoURL:      validRepoURL,
			commitHash:   validSHA,
			unsetEnv:     "PROJECT_ID",
			serverStatus: http.StatusOK,
			wantErr:      true,
		},
		{
			name:         "report POST rejected is an infrastructure failure",
			repoURL:      validRepoURL,
			commitHash:   validSHA,
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody compileResultRequest
			var gotReport bool

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotReport = true
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.WriteHeader(tt.serverStatus)
			}))
			defer srv.Close()

			env := map[string]string{
				"REPO_URL":         tt.repoURL,
				"GITHUB_TOKEN":     "fake-token",
				"COMMIT_HASH":      tt.commitHash,
				"PROJECT_ID":       "p-1",
				"ORGANIZATION_ID":  "org-1",
				"API_URL":          srv.URL,
				"CROSS_SERVER_JWT": "jwt-token",
			}
			for key, value := range env {
				if key == tt.unsetEnv {
					t.Setenv(key, "")
					continue
				}
				t.Setenv(key, value)
			}

			cmd := newCompileAndReportCmd()
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.RunE(cmd, nil)

			if tt.wantErr && err == nil {
				t.Fatal("expected a non-nil error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected exit 0 (report delivered), got error: %v", err)
			}

			if tt.wantReportedStatus != "" {
				if !gotReport {
					t.Fatal("expected a compile-result report to be posted, none was")
				}
				if gotBody.Status != tt.wantReportedStatus {
					t.Fatalf("reported status = %q, want %q", gotBody.Status, tt.wantReportedStatus)
				}
			}
		})
	}
}
