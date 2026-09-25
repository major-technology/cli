package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentEndpointsNameAgentAndOrganization(t *testing.T) {
	const agentID = "11111111-1111-4111-8111-111111111111"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing bearer token")
		}
		switch r.URL.Path {
		case "/cli/agents":
			if r.Method != "GET" || r.URL.Query().Get("organizationId") != "org-1" || r.URL.Query().Get("includeReadOnly") != "true" {
				t.Errorf("list request: %s", r.URL)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"agents": []any{map[string]any{
				"id": agentID, "name": "Helper", "description": "", "currentVersionId": "v1", "permissions": map[string]any{"canEdit": true},
			}}})
		case "/cli/agents/" + agentID + "/pull":
			if r.Method != "POST" {
				t.Errorf("pull method: %s", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"agentId": agentID, "name": "Helper", "version": 2, "downloadUrl": "https://example.com/get"})
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
	list, err := client.ListAgents("org-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Agents) != 1 || list.Agents[0].ID != agentID || list.Agents[0].CurrentVersionID == nil || !list.Agents[0].Permissions.CanEdit {
		t.Fatalf("list: %+v", list)
	}
	if resp, err := client.PullTarget("agents", agentID); err != nil || resp.Version != 2 {
		t.Fatalf("pull: %+v, %v", resp, err)
	}
}

type recordedRequest struct {
	method string
	path   string
	query  map[string][]string
	body   map[string]any
}

func agentRunServer(t *testing.T, response any) (*Client, *recordedRequest) {
	t.Helper()
	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.method = r.Method
		recorded.path = r.URL.Path
		recorded.query = r.URL.Query()
		recorded.body = nil
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&recorded.body)
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	previousToken := testTokenOverride
	testTokenOverride = "test-token"
	t.Cleanup(func() { testTokenOverride = previousToken })
	return NewClient(server.URL + "/cli"), recorded
}

const (
	testAgentID = "11111111-1111-4111-8111-111111111111"
	testRunID   = "22222222-2222-4222-8222-222222222222"
)

func TestGetAgentInfo(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{
		"id": testAgentID, "name": "Helper", "description": "", "publishedVersionId": nil,
		"latestVersion": map[string]any{"id": "v1", "version": 3, "notes": "n"},
		"envKeys":       []any{map[string]any{"key": "API_KEY", "hasValue": true}},
	})
	resp, err := client.GetAgentInfo(testAgentID)
	if err != nil {
		t.Fatal(err)
	}
	if req.method != "GET" || req.path != "/cli/agents/"+testAgentID {
		t.Fatalf("request: %s %s", req.method, req.path)
	}
	if resp.Name != "Helper" || resp.PublishedVersionID != nil || resp.LatestVersion.Version != 3 || !resp.EnvKeys[0].HasValue {
		t.Fatalf("response: %+v", resp)
	}
}

func TestStartAgentRun(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{"runId": testRunID, "status": "started"})
	resp, err := client.StartAgentRun(testAgentID, "Say hi", "Run 1")
	if err != nil {
		t.Fatal(err)
	}
	if req.method != "POST" || req.path != "/cli/agents/"+testAgentID+"/runs" {
		t.Fatalf("request: %s %s", req.method, req.path)
	}
	if req.body["prompt"] != "Say hi" || req.body["name"] != "Run 1" {
		t.Fatalf("body: %v", req.body)
	}
	if resp.RunID != testRunID || resp.Status != "started" {
		t.Fatalf("response: %+v", resp)
	}
	if _, err := client.StartAgentRun(testAgentID, "Say hi", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := req.body["name"]; ok {
		t.Fatalf("empty name sent: %v", req.body)
	}
}

func TestListAgentRunsWithEveryFilter(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{
		"runs": []any{map[string]any{
			"threadId": testRunID, "title": "Run 1", "user": nil, "source": "workflow", "channel": nil,
			"application": nil, "status": "idle", "pendingInputCount": 0, "createdAt": "c", "updatedAt": "u",
		}},
		"hasMore": true,
	})
	resp, err := client.ListAgentRuns(testAgentID, AgentRunListFilters{Mine: true, Live: true, Source: "workflow", Limit: 5, Offset: 10})
	if err != nil {
		t.Fatal(err)
	}
	if req.method != "GET" || req.path != "/cli/agents/"+testAgentID+"/runs" {
		t.Fatalf("request: %s %s", req.method, req.path)
	}
	want := map[string]string{"mine": "true", "live": "true", "source": "workflow", "limit": "5", "offset": "10"}
	for key, value := range want {
		if got := req.query[key]; len(got) != 1 || got[0] != value {
			t.Fatalf("query %s=%v", key, got)
		}
	}
	if !resp.HasMore || resp.Runs[0].Source != "workflow" {
		t.Fatalf("response: %+v", resp)
	}
}

func TestListAgentRunsWithoutFilters(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{"runs": []any{}, "hasMore": false})
	if _, err := client.ListAgentRuns(testAgentID, AgentRunListFilters{Limit: 20}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"mine", "live", "source"} {
		if _, ok := req.query[key]; ok {
			t.Fatalf("unexpected %s in query %v", key, req.query)
		}
	}
	if req.query["limit"][0] != "20" || req.query["offset"][0] != "0" {
		t.Fatalf("paging query %v", req.query)
	}
}

func TestSendAgentRunMessage(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{"status": "queued"})
	resp, err := client.SendAgentRunMessage(testRunID, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if req.method != "POST" || req.path != "/cli/agents/runs/"+testRunID+"/messages" || req.body["message"] != "hello" {
		t.Fatalf("request: %s %s %v", req.method, req.path, req.body)
	}
	if resp.Status != "queued" {
		t.Fatalf("response: %+v", resp)
	}
}

func TestStopAgentRun(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{"status": "stopped"})
	resp, err := client.StopAgentRun(testRunID)
	if err != nil {
		t.Fatal(err)
	}
	if req.method != "POST" || req.path != "/cli/agents/runs/"+testRunID+"/stop" {
		t.Fatalf("request: %s %s", req.method, req.path)
	}
	if resp.Status != "stopped" {
		t.Fatalf("response: %+v", resp)
	}
}

func TestGetAgentRunContent(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{"messages": []any{map[string]any{"id": "m1"}}, "nextToken": "t2"})
	resp, err := client.GetAgentRunContent(testRunID, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if req.method != "GET" || req.path != "/cli/agents/runs/"+testRunID+"/content" {
		t.Fatalf("request: %s %s", req.method, req.path)
	}
	if len(req.query) != 1 || req.query["nextToken"][0] != "t1" {
		t.Fatalf("query %v", req.query)
	}
	if len(resp.Messages) != 1 || resp.NextToken == nil || *resp.NextToken != "t2" {
		t.Fatalf("response: %+v", resp)
	}
	if _, err := client.GetAgentRunContent(testRunID, ""); err != nil {
		t.Fatal(err)
	}
	if len(req.query) != 0 {
		t.Fatalf("empty token sent: %v", req.query)
	}
}

func TestAgentSlackEndpoints(t *testing.T) {
	client, req := agentRunServer(t, map[string]any{"profileId": "p1", "installUrl": "https://slack.com/install"})
	connect, err := client.ConnectAgentSlack(testAgentID)
	if err != nil {
		t.Fatal(err)
	}
	if req.method != "POST" || req.path != "/cli/agents/"+testAgentID+"/slack/connect" || connect.InstallURL != "https://slack.com/install" {
		t.Fatalf("connect: %s %s %+v", req.method, req.path, connect)
	}

	client, req = agentRunServer(t, map[string]any{"success": true})
	for _, tc := range []struct {
		call   func(string) (*AgentSuccessResponse, error)
		method string
		path   string
	}{
		{client.PauseAgentSlack, "POST", "/cli/agents/" + testAgentID + "/slack/pause"},
		{client.ResumeAgentSlack, "POST", "/cli/agents/" + testAgentID + "/slack/resume"},
		{client.DeleteAgentSlack, "DELETE", "/cli/agents/" + testAgentID + "/slack"},
	} {
		resp, err := tc.call(testAgentID)
		if err != nil {
			t.Fatal(err)
		}
		if req.method != tc.method || req.path != tc.path || !resp.Success {
			t.Fatalf("%s %s: got %s %s %+v", tc.method, tc.path, req.method, req.path, resp)
		}
	}
}
