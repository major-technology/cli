package api

import "net/url"

// WorkflowItem is the web list's workflow item; the CLI decodes only what it prints.
type WorkflowItem struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	Description *string         `json:"description"`
	IsPublished bool            `json:"isPublished"`
	Permissions ListPermissions `json:"permissions"`
}
type WorkflowListResponse struct {
	Workflows []WorkflowItem `json:"workflows"`
}
type WorkflowCreateResponse struct {
	WorkflowID string `json:"workflowId"`
}

func (c *Client) ListWorkflows(organizationID string, includeReadOnly bool) (*WorkflowListResponse, error) {
	query := url.Values{"organizationId": {organizationID}}
	if includeReadOnly {
		query.Set("includeReadOnly", "true")
	}
	var resp WorkflowListResponse
	err := c.doRequest("GET", "/workflows?"+query.Encode(), nil, &resp)
	return &resp, err
}
func (c *Client) CreateWorkflow(organizationID string) (*WorkflowCreateResponse, error) {
	var resp WorkflowCreateResponse
	err := c.doRequest("POST", "/workflows", map[string]string{"organizationId": organizationID}, &resp)
	return &resp, err
}
