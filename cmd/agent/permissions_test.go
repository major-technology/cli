package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
)

func TestPermissionsNeedsExactlyOneTarget(t *testing.T) {
	for _, args := range [][]string{
		{"--id", testAgentID},
		{"--id", testAgentID, "--resource", "r", "--app", "a"},
	} {
		err := executeNonInteractive(newPermissionsCmd(), args...)
		if err == nil || !strings.Contains(err.Error(), "exactly one of --resource") {
			t.Fatalf("args=%v err=%v", args, err)
		}
	}
}

func TestPermissionsResultStringForResource(t *testing.T) {
	text := permissionsResult{permissions: []api.AgentPermission{
		{ToolName: "query", Decision: "allow", IsUpstream: true},
		{ToolName: "write", Decision: "deny"},
	}}.String()
	if text != "query  allow  (upstream)\nwrite  deny" {
		t.Fatalf("text: %q", text)
	}
}

func TestPermissionsResultStringForApp(t *testing.T) {
	text := permissionsResult{app: true, permissions: []api.AgentPermission{
		{ToolName: "t", Decision: "ask", Matcher: &api.AgentPermissionMatcher{Method: "GET", Path: "/x"}},
		{ToolName: "fallback", Decision: "allow"},
	}}.String()
	if text != "GET /x  ask\nfallback  allow" {
		t.Fatalf("text: %q", text)
	}
}

func TestPermissionsResultJSONIsTheArray(t *testing.T) {
	data, err := json.Marshal(permissionsResult{permissions: []api.AgentPermission{{ToolName: "query", Decision: "allow"}}})
	if err != nil || !strings.HasPrefix(string(data), `[{"toolName":"query"`) {
		t.Fatalf("data=%s err=%v", data, err)
	}
}
