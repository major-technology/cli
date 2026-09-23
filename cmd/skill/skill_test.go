package skill

import (
	"bytes"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/agent"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCommandsRegistered(t *testing.T) {
	for _, name := range []string{"list", "get", "create"} {
		if c, _, e := Cmd.Find([]string{name}); e != nil || c.Name() != name {
			t.Errorf("missing %s", name)
		}
	}
}

func TestGetSkillMatchingWorkspaceAndExplicitOverride(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)
	local := "11111111-1111-4111-8111-111111111111"
	explicit := "22222222-2222-4222-8222-222222222222"
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "skill", SkillID: local}}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	var path string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`{"id":"` + strings.TrimPrefix(path, "/skills/") + `"}`))
	}))
	defer s.Close()
	previous := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(s.URL))
	defer singletons.SetAPIClient(previous)
	g, _, _ := Cmd.Find([]string{"get"})
	var out bytes.Buffer
	g.SetOut(&out)
	for _, tc := range []struct {
		args []string
		want string
	}{{nil, local}, {[]string{explicit}, explicit}} {
		if err := g.RunE(g, tc.args); err != nil {
			t.Fatal(err)
		}
		if path != "/skills/"+tc.want {
			t.Fatalf("path %s", path)
		}
	}
	if _, err := agent.ResolveSkillID(""); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "app", ApplicationID: local}}); err != nil {
		t.Fatal(err)
	}
	if err := g.RunE(g, nil); err == nil {
		t.Fatal("accepted app target")
	}
}

func TestListPublishedCobraFlag(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.RequestURI() != "/skills?editable=true&published=true" {
			t.Errorf("request %s %s", r.Method, r.URL.RequestURI())
		}
		w.Write([]byte(`{"skills":[]}`))
	}))
	defer srv.Close()
	previous := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	defer singletons.SetAPIClient(previous)
	root := &cobra.Command{Use: "major", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(Cmd)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"skill", "list", "--published", "--editable", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"skills":[]`) {
		t.Fatal(out.String())
	}
}

func TestGetSkillExplicitIDOverridesWorkspaceCobra(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org", Target: workspace.Target{Kind: "skill", SkillID: "11111111-1111-4111-8111-111111111111"}}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/skills/explicit-skill" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"id":"explicit-skill"}`))
	}))
	defer srv.Close()
	previous := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(srv.URL))
	defer singletons.SetAPIClient(previous)
	root := &cobra.Command{Use: "major", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(Cmd)
	var out bytes.Buffer
	root.SetOut(&out)
	get, _, _ := Cmd.Find([]string{"get"})
	get.SetOut(&out)
	root.SetArgs([]string{"skill", "get", "explicit-skill", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id":"explicit-skill"`) {
		t.Fatal(out.String())
	}
}
