package api

import (
	"net/url"
)

type AgentItem struct {
	AgentID     string `json:"agentId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublished bool   `json:"isPublished"`
	CanEdit     bool   `json:"canEdit"`
}
type AgentListResponse struct {
	Agents []AgentItem `json:"agents"`
}
type AgentCreateResponse struct {
	AgentID string `json:"agentId"`
}

func (c *Client) ListAgents(organizationID string, includeReadOnly bool) (*AgentListResponse, error) {
	query := url.Values{"organizationId": {organizationID}}
	if includeReadOnly {
		query.Set("includeReadOnly", "true")
	}
	var resp AgentListResponse
	err := c.doRequest("GET", "/agents?"+query.Encode(), nil, &resp)
	return &resp, err
}
func (c *Client) CreateAgent(organizationID, name, description string) (*AgentCreateResponse, error) {
	var resp AgentCreateResponse
	err := c.doRequest("POST", "/agents", map[string]string{"organizationId": organizationID, "name": name, "description": description}, &resp)
	return &resp, err
}
