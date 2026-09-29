package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInspectEndpointsUseTheirPaths(t *testing.T) {
	const workflowID = "11111111-1111-4111-8111-111111111111"
	const runID = "22222222-2222-4222-8222-222222222222"
	const agentID = "33333333-3333-4333-8333-333333333333"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method: %s", r.Method)
		}
		switch r.URL.Path {
		case "/cli/workflows/" + workflowID + "/runs":
			_ = json.NewEncoder(w).Encode(map[string]any{"runs": []any{map[string]any{"id": runID, "status": "failed", "triggerType": "cron", "version": 3, "startedAt": "t0", "error": "boom"}}})
		case "/cli/workflows/" + workflowID + "/runs/" + runID:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"run":            map[string]any{"id": runID, "status": "completed", "version": 3},
				"document":       map[string]any{"run_id": runID, "executions": map[string]any{"a": map[string]any{"node_id": "a", "status": "completed", "output": map[string]any{"x": 1}}}},
				"definitionText": "{}",
			})
		case "/cli/workflows/connector-events/stripe":
			_ = json.NewEncoder(w).Encode(map[string]any{"connectorType": "stripe", "label": "Stripe", "resourceSubtype": "stripe", "events": []any{map[string]any{"type": "charge.succeeded", "label": "Charge", "payloadSchema": map[string]any{"type": "object"}}}})
		case "/cli/agents/" + agentID + "/resources/res-1/permissions":
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"resourceId": "res-1", "toolName": "query", "isUpstream": true, "decision": "allow"}})
		case "/cli/agents/" + agentID + "/applications/app-1/permissions":
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"applicationId": "app-1", "toolName": "t", "matcher": map[string]any{"method": "GET", "path": "/x"}, "isUpstream": false, "decision": "ask"}})
		case "/cli/applications/app-1/managed-database":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "active", "resourceId": "res-1", "message": "ready", "databaseId": nil})
		default:
			t.Errorf("unexpected URL %s", r.URL)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	previousToken := testTokenOverride
	testTokenOverride = "test-token"
	defer func() { testTokenOverride = previousToken }()
	client := NewClient(server.URL + "/cli")

	runs, err := client.ListWorkflowRuns(workflowID)
	if err != nil || len(runs.Runs) != 1 || runs.Runs[0].TriggerType != "cron" || runs.Runs[0].Version != 3 || runs.Runs[0].Error == nil || *runs.Runs[0].Error != "boom" {
		t.Fatalf("runs: %+v, %v", runs, err)
	}
	run, err := client.GetWorkflowRun(workflowID, runID)
	if err != nil || run.Run.Status != "completed" || run.Document == nil || run.Document.Executions["a"].Status != "completed" || run.DefinitionText != "{}" {
		t.Fatalf("run: %+v, %v", run, err)
	}
	events, err := client.GetWorkflowConnectorEvents("stripe")
	if err != nil || events.Label != "Stripe" || len(events.Events) != 1 || events.Events[0].PayloadSchema["type"] != "object" {
		t.Fatalf("events: %+v, %v", events, err)
	}
	resourcePerms, err := client.ListAgentResourcePermissions(agentID, "res-1")
	if err != nil || len(resourcePerms) != 1 || resourcePerms[0].ToolName != "query" || !resourcePerms[0].IsUpstream || resourcePerms[0].Decision != "allow" {
		t.Fatalf("resource perms: %+v, %v", resourcePerms, err)
	}
	appPerms, err := client.ListAgentApplicationPermissions(agentID, "app-1")
	if err != nil || len(appPerms) != 1 || appPerms[0].Matcher == nil || appPerms[0].Matcher.Path != "/x" || appPerms[0].Decision != "ask" {
		t.Fatalf("app perms: %+v, %v", appPerms, err)
	}
	database, err := client.GetManagedDatabaseStatus("app-1")
	if err != nil || database.Status != "active" || database.ResourceID == nil || *database.ResourceID != "res-1" || database.DatabaseID != nil || database.Message != "ready" {
		t.Fatalf("database: %+v, %v", database, err)
	}
}
