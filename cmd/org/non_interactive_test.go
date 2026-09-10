package org

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
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func TestSelectNonInteractiveRequiresIDWhenMultipleOrgs(t *testing.T) {
	keyring.MockInit()
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/organizations" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"organizations":[{"id":"o1","name":"One"},{"id":"o2","name":"Two"}]}`)
			return
		}
		writes.Add(1)
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })

	flagSelectOrgID = ""
	t.Cleanup(func() { flagSelectOrgID = "" })

	cmd := nonInteractiveOrgCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runSelect(cmd) })
	if err == nil {
		t.Fatal("org select without --id must not silently pick among multiple orgs")
	}
	if !strings.Contains(err.Error(), "--id") {
		t.Fatalf("error must name --id, got %v", err)
	}
	if _, _, orgErr := mjrToken.GetDefaultOrg(); orgErr == nil {
		t.Fatal("must not persist a default organization")
	}
	if writes.Load() != 0 {
		t.Fatalf("unexpected writes = %d", writes.Load())
	}
}

func nonInteractiveOrgCmd(t *testing.T) *cobra.Command {
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
