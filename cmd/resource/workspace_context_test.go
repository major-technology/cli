package resource

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

func TestRunListUsesSharedResolverNotFromRepo(t *testing.T) {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/applications/" + appID + "/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"applicationId":"`+appID+`","organizationId":"`+orgID+`","urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`)
		case "/resources":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"resources":[]}`)
		case "/applications/" + appID + "/resources":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"resources":[]}`)
		case "/application/from-repo":
			t.Errorf("resource list must not call /application/from-repo")
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(server.URL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runList(cmd); err != nil {
		t.Fatalf("runList() = %v", err)
	}
}
