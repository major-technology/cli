package resource

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
	"github.com/major-technology/cli/clients/config"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

const (
	niAppID = "11111111-1111-4111-8111-111111111111"
	niOrgID = "22222222-2222-4222-8222-222222222222"
)

func TestEnvNonInteractiveRequiresID(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: niOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: niAppID},
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/applications/"+niAppID+"/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID)
		case r.URL.Path == "/application/"+niAppID+"/environment" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"environmentId":"e1","environmentName":"dev"}`)
		case r.URL.Path == "/application/"+niAppID+"/environments":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"environments":[{"id":"e1","name":"dev"},{"id":"e2","name":"prod"}]}`)
		case r.URL.Path == "/application/"+niAppID+"/environment" && r.Method == http.MethodPost:
			writes.Add(1)
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

	flagEnvID = ""
	t.Cleanup(func() { flagEnvID = "" })

	cmd := nonInteractiveResourceCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runEnv(cmd) })
	if err == nil {
		t.Fatal("env without --id must fail")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--id") {
		t.Fatalf("error must name --id, got %v", err)
	}
	if !strings.Contains(msg, "env-list") && !strings.Contains(msg, "env list") {
		t.Fatalf("error must name env list --json discovery, got %v", err)
	}
	if writes.Load() != 0 {
		t.Fatalf("SetApplicationEnvironment calls = %d, want 0", writes.Load())
	}
}

func TestManageNonInteractiveRefusesAndNamesAddRemove(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: niOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: niAppID},
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/applications/"+niAppID+"/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID)
		case r.URL.Path == "/resources":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"resources":[{"id":"r1","name":"DB","type":"postgres","description":"db"}]}`)
		case r.URL.Path == "/application-resources":
			writes.Add(1)
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

	cmd := nonInteractiveResourceCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runManage(cmd) })
	if err == nil {
		t.Fatal("manage must refuse in non-interactive mode")
	}
	msg := err.Error()
	if !strings.Contains(msg, "add --id") {
		t.Fatalf("error must name resource add --id, got %v", err)
	}
	if !strings.Contains(msg, "remove --id") {
		t.Fatalf("error must name resource remove --id, got %v", err)
	}
	if writes.Load() != 0 {
		t.Fatalf("SaveApplicationResources calls = %d, want 0", writes.Load())
	}
}

func TestCreateNonInteractivePrintsURLWithoutOpening(t *testing.T) {
	orig := utils.BrowserStart
	opened := []string{}
	utils.BrowserStart = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	t.Cleanup(func() { utils.BrowserStart = orig })

	prevCfg := singletons.GetConfig()
	singletons.SetConfig(&config.Config{FrontendURI: "https://app.example.test"})
	t.Cleanup(func() { singletons.SetConfig(prevCfg) })

	cmd := nonInteractiveResourceCmd(t)
	err := runWithDeadline(t, 8*time.Second, func() error { return runCreate(cmd) })
	if err != nil {
		t.Fatalf("resource create: %v", err)
	}
	if len(opened) != 0 {
		t.Fatalf("opened browser: %v", opened)
	}
	out := cmd.OutOrStdout().(*bytes.Buffer).String()
	if !strings.Contains(out, "https://app.example.test/resources?action=add") {
		t.Fatalf("output must print destination, got %q", out)
	}
}

func nonInteractiveResourceCmd(t *testing.T) *cobra.Command {
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
