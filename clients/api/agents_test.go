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
			_ = json.NewEncoder(w).Encode(map[string]any{"agents": []any{}})
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
	if _, err := client.ListAgents("org-1", true); err != nil {
		t.Fatal(err)
	}
	if resp, err := client.PullAgent(agentID); err != nil || resp.Version != 2 {
		t.Fatalf("pull: %+v, %v", resp, err)
	}
}
