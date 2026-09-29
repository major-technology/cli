package workflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/spf13/cobra"
)

const testWorkflowID = "11111111-1111-4111-8111-111111111111"

func TestResolveWorkflowIDPrefersTheFlag(t *testing.T) {
	t.Chdir(t.TempDir())
	id, err := resolveWorkflowID("given")
	if err != nil || id != "given" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResolveWorkflowIDReadsTheWorkflowFolder(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "workflow", WorkflowID: testWorkflowID}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	id, err := resolveWorkflowID("")
	if err != nil || id != testWorkflowID {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResolveWorkflowIDRejectsANonWorkflowFolder(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "agent", AgentID: testWorkflowID}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	_, err := resolveWorkflowID("")
	if err == nil || !strings.Contains(err.Error(), "pass --id") || !strings.Contains(err.Error(), "workflow folder") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunIsAGroupWithListAndGet(t *testing.T) {
	cmd := newRunCmd()
	if cmd.RunE != nil || cmd.Run != nil || len(cmd.Commands()) != 2 {
		t.Fatalf("run group: %d subcommands", len(cmd.Commands()))
	}
}

func TestRunListResultString(t *testing.T) {
	if text := (runListResult{&api.WorkflowRunListResponse{}}).String(); text != "No runs found." {
		t.Fatalf("empty: %q", text)
	}
	boom := "boom"
	text := runListResult{&api.WorkflowRunListResponse{Runs: []api.WorkflowRun{
		{ID: "r1", Status: "failed", TriggerType: "cron", Version: 3, StartedAt: "t0", Error: &boom},
		{ID: "r2", Status: "completed", TriggerType: "manual", Version: 2, StartedAt: "t1"},
	}}}.String()
	want := "r1  failed  cron  v3  t0  boom\nr2  completed  manual  v2  t1"
	if text != want {
		t.Fatalf("text: %q", text)
	}
}

func TestRunGetResultString(t *testing.T) {
	noTrace := runGetResult{&api.WorkflowRunResponse{Run: api.WorkflowRun{ID: "r1", Status: "running", Version: 1}}}.String()
	if !strings.Contains(noTrace, "r1  running  v1") || !strings.Contains(noTrace, "not started writing its trace") {
		t.Fatalf("no trace: %q", noTrace)
	}
	nodeErr := "bad input"
	text := runGetResult{&api.WorkflowRunResponse{
		Run: api.WorkflowRun{ID: "r1", Status: "failed", Version: 1},
		Document: &api.WorkflowRunDocument{Executions: map[string]api.WorkflowRunExecution{
			"b": {Status: "failed", TriggeredAt: "2", Error: &nodeErr},
			"a": {Status: "completed", TriggeredAt: "1", Output: map[string]any{"x": 1}},
		}},
	}}.String()
	want := "r1  failed  v1\na  completed\n  output: {\"x\":1}\nb  failed  bad input"
	if text != want {
		t.Fatalf("text: %q", text)
	}
}

func TestEventsResultString(t *testing.T) {
	text := eventsResult{&api.WorkflowConnectorProvider{ConnectorType: "stripe", Label: "Stripe", Events: []api.WorkflowConnectorEvent{{Type: "charge.succeeded", Label: "Charge", Description: "A charge"}}}}.String()
	if text != "stripe  Stripe\ncharge.succeeded  Charge  A charge" {
		t.Fatalf("text: %q", text)
	}
	one := eventResult{&api.WorkflowConnectorEvent{Type: "charge.succeeded", Label: "Charge", PayloadSchema: map[string]any{"type": "object"}}}.String()
	if !strings.Contains(one, "\n  \"payloadSchema\"") {
		t.Fatalf("event: %q", one)
	}
}

func TestEventsNeedsAConnectorType(t *testing.T) {
	var out bytes.Buffer
	var cmd *cobra.Command = newEventsCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an arg error")
	}
}
