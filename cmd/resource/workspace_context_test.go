package resource

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

func TestRunListDoesNotRequireAnApplication(t *testing.T) {
	const (
		appID = "11111111-1111-4111-8111-111111111111"
		orgID = "22222222-2222-4222-8222-222222222222"
	)
	for _, target := range []string{"app", "agent", "none"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			if target != "none" {
				cfg := workspace.Config{OrganizationID: orgID, Target: workspace.Target{Kind: target}}
				if target == "app" {
					cfg.Target.ApplicationID = appID
				} else {
					cfg.Target.AgentID = appID
				}
				if err := workspace.Write(dir, cfg); err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(dir)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/resources" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"resources":[{"id":"resource-1","name":"Database","type":"postgres","description":"Test"}]}`)
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
			var resources []map[string]any
			if err := json.Unmarshal(out.Bytes(), &resources); err != nil {
				t.Fatal(err)
			}
			if len(resources) != 1 || resources[0]["id"] != "resource-1" {
				t.Fatalf("unexpected resources: %v", resources)
			}
			if _, exists := resources[0]["isAttached"]; exists {
				t.Fatalf("obsolete isAttached field in %v", resources[0])
			}
		})
	}
}
