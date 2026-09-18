package vars

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

func TestRequestSecretSetup(t *testing.T) {
	for _, delivery := range []string{"thread", "url"} {
		t.Run(delivery, func(t *testing.T) {
			const appID = "11111111-1111-4111-8111-111111111111"
			dir := t.TempDir()
			if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "app", ApplicationID: appID}}); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)
			t.Setenv("MAJOR_TOKEN", "test-token")
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/applications/" + appID + "/info":
					fmt.Fprintf(w, `{"applicationId":%q,"organizationId":"org-1"}`, appID)
				case "/application/" + appID + "/env-variables/request-setup":
					called = true
					var body struct {
						Kind string
						Keys []string
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if r.Method != "POST" || body.Kind != "env" || strings.Join(body.Keys, ",") != "API_KEY" {
						t.Errorf("unexpected request: %+v", body)
					}
					fmt.Fprintf(w, `{"delivery":%q,"message":"setup response"}`, delivery)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			previous := singletons.GetAPIClient()
			singletons.SetAPIClient(api.NewClient(server.URL))
			t.Cleanup(func() { singletons.SetAPIClient(previous) })
			cmd := &cobra.Command{}
			output := new(bytes.Buffer)
			cmd.SetOut(output)
			if err := runRequest(cmd, []string{"API_KEY"}, false); err != nil {
				t.Fatal(err)
			}
			if !called || output.String() != "setup response\n" {
				t.Fatalf("called=%v output=%q", called, output.String())
			}
		})
	}
}
