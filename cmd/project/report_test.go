package project

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
