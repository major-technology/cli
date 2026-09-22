package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentSkillRoutes(t *testing.T) {
	previous := testTokenOverride
	testTokenOverride = "test-token"
	defer func() { testTokenOverride = previous }()
	cases := []struct {
		name, method, path, body string
		call                     func(*Client) error
	}{
		{"agents", "GET", "/agents?editable=true", "", func(c *Client) error { _, e := c.ListAgents(true); return e }},
		{"run", "POST", "/agents/a%2Fb/runs", `{"prompt":"hello"}`, func(c *Client) error { _, e := c.StartAgentRun("a/b", "hello", ""); return e }},
		{"runs", "GET", "/agent-runs?allUsers=true", "", func(c *Client) error { _, e := c.ListAgentRuns("", true); return e }},
		{"content", "GET", "/agent-runs/a%2Fb/content?n=3", "", func(c *Client) error { _, e := c.GetAgentRunContent("a/b", 3); return e }},
		{"send", "POST", "/agent-runs/r/messages", `{"message":"hi"}`, func(c *Client) error { _, e := c.SendAgentRunMessage("r", "hi"); return e }},
		{"channel", "DELETE", "/agents/a/channel", `{"type":"slack"}`, func(c *Client) error { _, e := c.AgentChannel("a", "delete"); return e }},
		{"skills", "GET", "/skills?published=true", "", func(c *Client) error { _, e := c.ListSkills(false, true); return e }},
		{"apps", "GET", "/apps?editable=true", "", func(c *Client) error { _, e := c.ListApps(true); return e }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := http.StatusOK
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.EscapedPath()+func() string {
					if r.URL.RawQuery != "" {
						return "?" + r.URL.RawQuery
					}
					return ""
				}() != tc.path {
					t.Errorf("got %s %s", r.Method, r.URL.String())
				}
				b := make([]byte, 1024)
				n, _ := r.Body.Read(b)
				if string(b[:n]) != tc.body {
					t.Errorf("body %q want %q", b[:n], tc.body)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing auth")
				}
				w.WriteHeader(status)
				w.Write([]byte(`{}`))
			}))
			defer s.Close()
			c := NewClient(s.URL)
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
			status = 403
			if err := tc.call(c); err == nil || strings.Contains(err.Error(), "test-token") {
				t.Fatalf("expected safe HTTP error, got %v", err)
			}
		})
	}
}
