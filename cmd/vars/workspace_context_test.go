package vars

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
)

func TestGetAppIDUsesSharedResolverNotFromRepo(t *testing.T) {
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
		if r.URL.Path == "/application/from-repo" {
			t.Errorf("vars must not call /application/from-repo")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/applications/"+appID+"/info" {
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"applicationId":"`+appID+`","organizationId":"`+orgID+`","urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`)
	}))
	defer server.Close()

	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(server.URL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })

	got, err := getAppID()
	if err != nil {
		t.Fatalf("getAppID() = %v", err)
	}
	if got != appID {
		t.Fatalf("getAppID() = %q, want %q", got, appID)
	}
}
