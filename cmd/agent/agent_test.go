package agent

import (
	"bytes"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCommandsRegistered(t *testing.T) {
	for _, path := range []string{"list", "get", "create", "run", "runs list", "runs content", "runs send", "runs stop", "channel connect", "channel pause", "channel resume", "channel delete", "permissions resource", "permissions app"} {
		found, _, err := Cmd.Find(strings.Fields(path))
		parts := strings.Fields(path)
		if err != nil || found.Name() != parts[len(parts)-1] {
			t.Errorf("missing %s: %v", path, err)
		} else if found.Flags().Lookup("json") == nil {
			t.Errorf("missing --json for %s", path)
		}
	}
}

func TestTargetUsesExplicitIDWithoutCheckout(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if got, err := target("explicit", "agent"); err != nil || got != "explicit" {
		t.Fatalf("got %q: %v", got, err)
	}
	if _, err := target("", "agent"); err == nil {
		t.Fatal("missing ID accepted")
	}
}
func TestTargetUsesMatchingWorkspaceOnly(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	cfg := workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "agent", AgentID: "11111111-1111-4111-8111-111111111111"}}
	if err := workspace.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	if got, err := target("", "agent"); err != nil || got != cfg.Target.AgentID {
		t.Fatalf("got %q: %v", got, err)
	}
	if got, err := target("explicit", "agent"); err != nil || got != "explicit" {
		t.Fatalf("override %q: %v", got, err)
	}
	if _, err := target("", "skill"); err == nil {
		t.Fatal("mismatched target accepted")
	}
}

func TestGetUsesWorkspaceAndExplicitOverrideJSON(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)
	local := "11111111-1111-4111-8111-111111111111"
	explicit := "22222222-2222-4222-8222-222222222222"
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "agent", AgentID: local}}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	var requested string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.Path
		w.Write([]byte(`{"id":"` + strings.TrimPrefix(requested, "/agents/") + `","name":"Test"}`))
	}))
	defer s.Close()
	previous := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(s.URL))
	defer singletons.SetAPIClient(previous)
	for _, tc := range []struct {
		args []string
		want string
	}{{[]string{"--json"}, local}, {[]string{explicit, "--json"}, explicit}} {
		var out bytes.Buffer
		get := Cmd.Commands()[0]
		for _, c := range Cmd.Commands() {
			if c.Name() == "get" {
				get = c
				break
			}
		}
		get.SetOut(&out)
		get.Flags().Set("json", "true")
		args := tc.args[:len(tc.args)-1]
		if err := get.RunE(get, args); err != nil {
			t.Fatal(err)
		}
		if requested != "/agents/"+tc.want || !strings.Contains(out.String(), `"name":"Test"`) {
			t.Fatalf("request %s output %s", requested, out.String())
		}
	}
}
