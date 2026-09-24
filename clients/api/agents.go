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
type AgentPullResponse struct {
	AgentID     string `json:"agentId"`
	Name        string `json:"name"`
	Version     int    `json:"version"`
	DownloadURL string `json:"downloadUrl"`
}
type AgentUploadResponse struct {
	UploadURL string `json:"uploadUrl"`
	UploadKey string `json:"uploadKey"`
}
type AgentPushResponse struct {
	AgentID string `json:"agentId"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}
type AgentValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}
type AgentValidateResponse struct {
	Valid    bool                   `json:"valid"`
	Errors   []AgentValidationError `json:"errors"`
	Warnings []string               `json:"warnings"`
}
type AgentPublishResponse struct {
	VersionID string `json:"versionId"`
	Version   int    `json:"version"`
}
type AgentFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
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
func agentPath(id, action string) string { return "/agents/" + url.PathEscape(id) + "/" + action }
func (c *Client) PullAgent(id string) (*AgentPullResponse, error) {
	var resp AgentPullResponse
	err := c.doRequest("POST", agentPath(id, "pull"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) AgentUploadURL(id string) (*AgentUploadResponse, error) {
	var resp AgentUploadResponse
	err := c.doRequest("POST", agentPath(id, "push-upload-url"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) PushAgent(id, key, notes string) (*AgentPushResponse, error) {
	var resp AgentPushResponse
	err := c.doRequest("POST", agentPath(id, "push"), map[string]string{"uploadKey": key, "notes": notes}, &resp)
	return &resp, err
}
func (c *Client) ValidateAgent(id string, files []AgentFile) (*AgentValidateResponse, error) {
	var resp AgentValidateResponse
	err := c.doRequest("POST", agentPath(id, "validate"), map[string]any{"files": files}, &resp)
	return &resp, err
}
func (c *Client) PublishAgent(id string) (*AgentPublishResponse, error) {
	var resp AgentPublishResponse
	err := c.doRequest("POST", agentPath(id, "publish"), map[string]any{}, &resp)
	return &resp, err
}
