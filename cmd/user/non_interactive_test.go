package user

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	apiClient "github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func TestLoginNonInteractiveFailsImmediately(t *testing.T) {
	keyring.MockInit()
	t.Setenv("MAJOR_TOKEN", "")

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		t.Errorf("unexpected HTTP call: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(apiClient.NewClient(srv.URL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })

	cmd := nonInteractiveUserCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runLogin(cmd) })
	if err == nil {
		t.Fatal("non-interactive login must fail")
	}
	if !strings.Contains(err.Error(), "MAJOR_TOKEN") {
		t.Fatalf("error must name MAJOR_TOKEN, got %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("HTTP calls = %d, want 0", got)
	}
}

func TestLinkLoginNonInteractiveFailsImmediately(t *testing.T) {
	keyring.MockInit()
	t.Setenv("MAJOR_TOKEN", "")
	singletons.SetAPIClient(apiClient.NewClient("http://127.0.0.1:1"))

	cmd := nonInteractiveUserCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return RunLoginForLink(cmd) })
	if err == nil {
		t.Fatal("non-interactive implicit link login must fail")
	}
	if !strings.Contains(err.Error(), "MAJOR_TOKEN") {
		t.Fatalf("error must name MAJOR_TOKEN, got %v", err)
	}
}

func TestGitconfigNonInteractiveRequiresUsername(t *testing.T) {
	keyring.MockInit()
	flagGitconfigUsername = ""
	t.Cleanup(func() { flagGitconfigUsername = "" })

	cmd := nonInteractiveUserCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runGitConfig(cmd) })
	if err == nil {
		t.Fatal("gitconfig without --username must fail")
	}
	if !strings.Contains(err.Error(), "--username") {
		t.Fatalf("error must name --username, got %v", err)
	}
}

func nonInteractiveUserCmd(t *testing.T) *cobra.Command {
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
