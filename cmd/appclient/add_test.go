package appclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

const (
	testOrgID    = "11111111-1111-4111-8111-111111111111"
	testAppID    = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testTargetID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func prepare(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: testOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: testAppID},
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("MAJOR_TOKEN", "test-injected-token")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/applications/" + testAppID + "/info":
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"a","name":"A","deployStatus":"deployed","appUrl":null}`, testAppID, testOrgID)
		case "/organizations/applications":
			fmt.Fprintf(w, `{"applications":[{"id":%q,"name":"Order Service"}]}`, testTargetID)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })

	origEnsure := ensurePackage
	ensurePackage = func(*cobra.Command, string) error { return nil }
	t.Cleanup(func() { ensurePackage = origEnsure })
}

func TestAddGeneratesClientForAppInOrg(t *testing.T) {
	prepare(t)

	var gotArgs []string
	orig := runAppClientCLI
	runAppClientCLI = func(cmd *cobra.Command, dir string, args ...string) error {
		gotArgs = args
		return nil
	}
	t.Cleanup(func() { runAppClientCLI = orig })

	flagAddID, flagAddName, flagAddJSON = testTargetID, "", true
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := runAdd(cmd); err != nil {
		t.Fatal(err)
	}

	if len(gotArgs) < 3 || gotArgs[0] != "add" || gotArgs[1] != testTargetID || gotArgs[2] != "Order Service" {
		t.Fatalf("args = %v", gotArgs)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("stdout not JSON: %q", out.String())
	}
	if got["appId"] != testTargetID || got["added"] != true {
		t.Fatalf("json = %v", got)
	}
}

func TestAddRejectsAppOutsideOrg(t *testing.T) {
	prepare(t)
	flagAddID, flagAddName, flagAddJSON = "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "", false

	if err := runAdd(&cobra.Command{}); err == nil {
		t.Fatal("expected an error for an app outside the org")
	}
}
