package api

import (
	"net/url"
)

type ListPermissions struct {
	CanEdit bool `json:"canEdit"`
}

// AgentItem is the web list's agent summary; the CLI decodes only what it prints.
type AgentItem struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	CurrentVersionID *string         `json:"currentVersionId"`
	Permissions      ListPermissions `json:"permissions"`
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
