package api

import (
	"net/url"
	"strconv"
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
type AgentInfoResponse struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	CurrentVersionID *string             `json:"currentVersionId"`
	LatestVersion    *AgentLatestVersion `json:"latestVersion"`
}
type AgentLatestVersion struct {
	ID        string  `json:"id"`
	AgentID   string  `json:"agentId"`
	Version   int     `json:"version"`
	CreatedBy string  `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
	Notes     *string `json:"notes"`
}
type AgentRunStartResponse struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
}
type AgentRunStatusResponse struct {
	Status string `json:"status"`
}
type AgentRunSummary struct {
	RunID             string           `json:"runId"`
	Title             string           `json:"title"`
	User              *AgentRunUser    `json:"user"`
	Source            string           `json:"source"`
	Channel           *string          `json:"channel"`
	Application       *AgentRunRelated `json:"application"`
	Status            string           `json:"status"`
	PendingInputCount int              `json:"pendingInputCount"`
	CreatedAt         string           `json:"createdAt"`
	UpdatedAt         string           `json:"updatedAt"`
}
type AgentRunUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type AgentRunRelated struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type AgentRunListResponse struct {
	Runs    []AgentRunSummary `json:"runs"`
	HasMore bool              `json:"hasMore"`
}
type AgentRunListFilters struct {
	Mine   bool
	Live   bool
	Source string
	Limit  int
	Offset int
}
type AgentRunMessage struct {
	Role      string `json:"role"`
	Type      string `json:"type"`
	Content   any    `json:"content"`
	Timestamp string `json:"timestamp"`
}
type AgentRunMessagesResponse struct {
	Messages  []AgentRunMessage `json:"messages"`
	NextToken *string           `json:"nextToken,omitempty"`
}
type AgentSlackConnectResponse struct {
	ProfileID  string `json:"profileId"`
	InstallURL string `json:"installUrl"`
}
type AgentSuccessResponse struct {
	Success bool `json:"success"`
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
func (c *Client) GetAgentInfo(id string) (*AgentInfoResponse, error) {
	var resp AgentInfoResponse
	err := c.doRequest("GET", "/agents/"+url.PathEscape(id), nil, &resp)
	return &resp, err
}
func (c *Client) StartAgentRun(id, prompt, name string) (*AgentRunStartResponse, error) {
	body := map[string]string{"prompt": prompt}
	if name != "" {
		body["name"] = name
	}
	var resp AgentRunStartResponse
	err := c.doRequest("POST", agentPath(id, "runs"), body, &resp)
	return &resp, err
}
func (c *Client) ListAgentRuns(id string, filters AgentRunListFilters) (*AgentRunListResponse, error) {
	query := url.Values{"limit": {strconv.Itoa(filters.Limit)}, "offset": {strconv.Itoa(filters.Offset)}}
	if filters.Mine {
		query.Set("mine", "true")
	}
	if filters.Live {
		query.Set("live", "true")
	}
	if filters.Source != "" {
		query.Set("source", filters.Source)
	}
	var resp AgentRunListResponse
	err := c.doRequest("GET", agentPath(id, "runs")+"?"+query.Encode(), nil, &resp)
	return &resp, err
}
func runPath(runID, action string) string {
	return "/agents/runs/" + url.PathEscape(runID) + "/" + action
}
func (c *Client) SendAgentRunMessage(runID, message string) (*AgentRunStatusResponse, error) {
	var resp AgentRunStatusResponse
	err := c.doRequest("POST", runPath(runID, "messages"), map[string]string{"message": message}, &resp)
	return &resp, err
}
func (c *Client) StopAgentRun(runID string) (*AgentRunStatusResponse, error) {
	var resp AgentRunStatusResponse
	err := c.doRequest("POST", runPath(runID, "stop"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) GetAgentRunMessages(runID, nextToken string) (*AgentRunMessagesResponse, error) {
	path := runPath(runID, "messages")
	if nextToken != "" {
		path += "?" + url.Values{"nextToken": {nextToken}}.Encode()
	}
	var resp AgentRunMessagesResponse
	err := c.doRequest("GET", path, nil, &resp)
	return &resp, err
}
func (c *Client) ConnectAgentSlack(id string) (*AgentSlackConnectResponse, error) {
	var resp AgentSlackConnectResponse
	err := c.doRequest("POST", agentPath(id, "slack/connect"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) PauseAgentSlack(id string) (*AgentSuccessResponse, error) {
	var resp AgentSuccessResponse
	err := c.doRequest("POST", agentPath(id, "slack/pause"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) ResumeAgentSlack(id string) (*AgentSuccessResponse, error) {
	var resp AgentSuccessResponse
	err := c.doRequest("POST", agentPath(id, "slack/resume"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) DeleteAgentSlack(id string) (*AgentSuccessResponse, error) {
	var resp AgentSuccessResponse
	err := c.doRequest("DELETE", agentPath(id, "slack"), nil, &resp)
	return &resp, err
}
