package agent

import (
	"bytes"
	"encoding/json"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
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

// executeAgent exercises Cobra flag parsing and required-flag handling, not RunE directly.
func executeAgent(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "major", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(Cmd)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"agent"}, args...))
	err := root.Execute()
	return out.String(), err
}

func TestContentRejectsExplicitNonpositiveLimit(t *testing.T) {
	content, _, _ := Cmd.Find([]string{"runs", "content"})
	limit := content.Flags().Lookup("limit")
	limit.Changed = false
	if err := limit.Value.Set("0"); err != nil {
		t.Fatal(err)
	}
	defer func() { limit.Changed = false; _ = limit.Value.Set("0") }()
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.Write([]byte(`{"messages":[]}`)) }))
	defer srv.Close()
	old := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	defer singletons.SetAPIClient(old)
	out, err := executeAgent(t, "runs", "content", "run-id", "--json")
	if err != nil || !strings.Contains(out, `"messages":[]`) {
		t.Fatalf("omitted limit output %q: %v", out, err)
	}
	for _, value := range []string{"0", "-1"} {
		_, err := executeAgent(t, "runs", "content", "run-id", "--limit", value, "--json")
		if err == nil || !strings.Contains(err.Error(), "limit must be positive") {
			t.Errorf("--limit %s: %v", value, err)
		}
	}
	if calls != 1 {
		t.Fatalf("invalid limits sent requests; total %d", calls)
	}
}

func TestRunListAllUsersCobraFlag(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/agent-runs?agentId=target&allUsers=true" {
			t.Errorf("request %s", r.URL.RequestURI())
		}
		w.Write([]byte(`{"runs":[],"hasMore":false}`))
	}))
	defer srv.Close()
	old := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	defer singletons.SetAPIClient(old)
	out, err := executeAgent(t, "runs", "list", "--agent", "target", "--all-users", "--json")
	if err != nil || !strings.Contains(out, `"hasMore":false`) {
		t.Fatalf("output %q: %v", out, err)
	}
}

func TestRunPromptAndWorkspaceIDOverrideCobraFlags(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)
	local := "11111111-1111-4111-8111-111111111111"
	explicit := "22222222-2222-4222-8222-222222222222"
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "agent", AgentID: local}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/agents/"+explicit+"/runs" || r.Method != "POST" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["prompt"] != "say hello" || body["name"] != "demo" {
			t.Errorf("body %+v", body)
		}
		w.Write([]byte(`{"chatThreadId":"run","status":"started"}`))
	}))
	defer srv.Close()
	old := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	defer singletons.SetAPIClient(old)
	out, err := executeAgent(t, "run", explicit, "--prompt", "say hello", "--name", "demo", "--json")
	if err != nil || !strings.Contains(out, `"status":"started"`) {
		t.Fatalf("output %q: %v", out, err)
	}
	if calls != 1 {
		t.Fatalf("calls %d", calls)
	}
}

func TestChannelTypeAndWorkspaceIDOverrideCobraFlags(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)
	local := "11111111-1111-4111-8111-111111111111"
	explicit := "22222222-2222-4222-8222-222222222222"
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "agent", AgentID: local}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/agents/"+explicit+"/channel/pause" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["type"] != "slack" {
			t.Errorf("body %+v", body)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	old := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	defer singletons.SetAPIClient(old)
	out, err := executeAgent(t, "channel", "pause", explicit, "--type", "slack", "--json")
	if err != nil || !strings.Contains(out, `"ok":true`) {
		t.Fatalf("output %q: %v", out, err)
	}
	if calls != 1 {
		t.Fatalf("calls %d", calls)
	}
}

func TestPermissionsResourceExplicitIDOverridesWorkspaceCobra(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "agent", AgentID: "11111111-1111-4111-8111-111111111111"}}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/agents/explicit-agent/permissions/resources/resource-id" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"resourcePermissions":[]}`))
	}))
	defer srv.Close()
	old := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	defer singletons.SetAPIClient(old)
	out, err := executeAgent(t, "permissions", "resource", "explicit-agent", "resource-id", "--json")
	if err != nil || !strings.Contains(out, `"resourcePermissions":[]`) {
		t.Fatalf("output %q: %v", out, err)
	}
}
