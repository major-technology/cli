package api

import (
	"fmt"
	"net/http"
	"net/url"
)

type Record = map[string]any

func (c *Client) agentRequest(method, path string, body any) (Record, error) {
	var out Record
	err := c.doRequest(method, path, body, &out)
	return out, err
}
func agentPath(id string) string { return "/agents/" + url.PathEscape(id) }
func runPath(id string) string   { return "/agent-runs/" + url.PathEscape(id) }
func (c *Client) ListAgents(editable bool) (Record, error) {
	p := "/agents"
	if editable {
		p += "?editable=true"
	}
	return c.agentRequest("GET", p, nil)
}
func (c *Client) GetAgent(id string) (Record, error) {
	return c.agentRequest("GET", agentPath(id), nil)
}
func (c *Client) CreateAgent(name, description string) (Record, error) {
	body := Record{"name": name}
	if description != "" {
		body["description"] = description
	}
	return c.agentRequest("POST", "/agents", body)
}
func (c *Client) StartAgentRun(id, prompt, name string) (Record, error) {
	body := Record{"prompt": prompt}
	if name != "" {
		body["name"] = name
	}
	return c.agentRequest("POST", agentPath(id)+"/runs", body)
}
func (c *Client) ListAgentRuns(agentID string, allUsers bool) (Record, error) {
	q := url.Values{}
	if agentID != "" {
		q.Set("agentId", agentID)
	}
	if allUsers {
		q.Set("allUsers", "true")
	}
	p := "/agent-runs"
	if len(q) > 0 {
		p += "?" + q.Encode()
	}
	return c.agentRequest("GET", p, nil)
}
func (c *Client) GetAgentRunContent(id string, limit int) (Record, error) {
	p := runPath(id) + "/content"
	if limit > 0 {
		p += fmt.Sprintf("?n=%d", limit)
	}
	return c.agentRequest("GET", p, nil)
}
func (c *Client) SendAgentRunMessage(id, message string) (Record, error) {
	return c.agentRequest("POST", runPath(id)+"/messages", Record{"message": message})
}
func (c *Client) StopAgentRun(id string) (Record, error) {
	return c.agentRequest("POST", runPath(id)+"/stop", nil)
}
func (c *Client) AgentChannel(id, action string) (Record, error) {
	method := http.MethodPost
	p := agentPath(id) + "/channel"
	if action != "delete" {
		p += "/" + action
	} else {
		method = http.MethodDelete
	}
	return c.agentRequest(method, p, Record{"type": "slack"})
}
func (c *Client) AgentResourcePermissions(agentID, resourceID string) (Record, error) {
	return c.agentRequest("GET", agentPath(agentID)+"/permissions/resources/"+url.PathEscape(resourceID), nil)
}
func (c *Client) AgentAppPermissions(agentID, appID string) (Record, error) {
	return c.agentRequest("GET", agentPath(agentID)+"/permissions/apps/"+url.PathEscape(appID), nil)
}
