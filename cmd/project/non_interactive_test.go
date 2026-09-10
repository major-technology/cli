package project

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

func TestDeployNonInteractiveRequiresYesBeforeDelete(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "remote", "add", "origin", "https://github.com/acme/demo.git")
	t.Chdir(dir)
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	var deploys atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/from-repo":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"projectId":"p-1","organizationId":"org-1"}`)
		case strings.HasPrefix(r.URL.Path, "/projects/p-1/versions"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"versions":[{"id":"v-1","commitHash":"aaaaaaaaaaaa","compileStatus":"compiled","createdAt":"2026-09-09T00:00:00Z"}]}`)
		case strings.Contains(r.URL.Path, "/deploy-plan"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"creates":[],"updates":[],"unchanged":[],"deletes":["old-agent"]}`)
		case strings.Contains(r.URL.Path, "/deploys"):
			deploys.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })

	cmd := nonInteractiveProjectCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runDeploy(cmd, "", false) })
	if err == nil {
		t.Fatal("project deploy deletions require --yes even with --non-interactive")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("error must name --yes, got %v", err)
	}
	if deploys.Load() != 0 {
		t.Fatalf("CreateProjectDeploy calls = %d, want 0", deploys.Load())
	}
}

func nonInteractiveProjectCmd(t *testing.T) *cobra.Command {
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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
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
