package vars

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

func TestUnsetNonInteractiveRequiresYes(t *testing.T) {
	const (
		appID = "11111111-1111-4111-8111-111111111111"
		orgID = "22222222-2222-4222-8222-222222222222"
	)
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: orgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: appID},
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	var deletes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/applications/"+appID+"/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, appID, orgID)
		case strings.Contains(r.URL.Path, "/env-variables/by-key/"):
			deletes.Add(1)
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

	flagUnsetYes = false
	t.Cleanup(func() {
		flagUnsetYes = false
	})

	cmd := nonInteractiveVarsCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runUnset(cmd, "CLI_PROTOTYPE") })
	if err == nil {
		t.Fatal("--non-interactive must not imply --yes")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("error must name --yes, got %v", err)
	}
	if deletes.Load() != 0 {
		t.Fatalf("delete calls = %d, want 0", deletes.Load())
	}
}

func nonInteractiveVarsCmd(t *testing.T) *cobra.Command {
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
