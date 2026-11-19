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

// WorkflowPull is a workflow's latest saved version: its definition, and the
// archive of every other workspace file when the version has one.
type WorkflowPull struct {
	TargetVersion
	ArchiveDownloadURL *string `json:"functionsDownloadUrl,omitempty"`
}

// WorkflowUpload is where a push PUTs the definition and the archive of the
// rest of the workspace. An API that stores no archive leaves ArchiveURL empty.
type WorkflowUpload struct {
	UploadURL  string `json:"uploadUrl"`
	UploadKey  string `json:"uploadKey"`
	ArchiveURL string `json:"functionsUploadUrl"`
	ArchiveKey string `json:"functionsUploadKey"`
}

func (c *Client) PullWorkflow(id string) (*WorkflowPull, error) {
	var resp WorkflowPull
	err := c.doRequest("POST", targetPath("workflows", id, "pull"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) WorkflowUploadURL(id string) (*WorkflowUpload, error) {
	var resp WorkflowUpload
	err := c.doRequest("POST", targetPath("workflows", id, "push-upload-url"), map[string]any{}, &resp)
	return &resp, err
}

// PushWorkflow saves an uploaded definition, with the archive when one was uploaded.
func (c *Client) PushWorkflow(id, definitionKey, archiveKey string) (*TargetVersion, error) {
	body := map[string]string{"uploadKey": definitionKey}
	if archiveKey != "" {
		body["functionsUploadKey"] = archiveKey
	}
	var resp TargetVersion
	err := c.doRequest("POST", targetPath("workflows", id, "push"), body, &resp)
	return &resp, err
}

type WorkflowRun struct {
	ID                string  `json:"id"`
	WorkflowID        string  `json:"workflowId"`
	Status            string  `json:"status"`
	TriggerType       string  `json:"triggerType"`
	TriggerID         *string `json:"triggerId"`
	WorkflowVersionID string  `json:"workflowVersionId"`
	Version           int     `json:"version"`
	ScheduledAt       *string `json:"scheduledAt"`
	StartedAt         string  `json:"startedAt"`
	CompletedAt       *string `json:"completedAt"`
	Error             *string `json:"error"`
	SingleNodeID      *string `json:"singleNodeId"`
	CreatedBy         string  `json:"createdBy"`
}
type WorkflowRunListResponse struct {
	Runs []WorkflowRun `json:"runs"`
}
type WorkflowRunExecution struct {
	NodeID      string  `json:"node_id"`
	Frame       string  `json:"frame"`
	Status      string  `json:"status"`
	TriggeredAt string  `json:"triggered_at"`
	CompletedAt *string `json:"completed_at,omitempty"`
	Input       any     `json:"input,omitempty"`
	Output      any     `json:"output,omitempty"`
	Error       *string `json:"error,omitempty"`
	ChatThread  *string `json:"chat_thread_id,omitempty"`
	Items       []any   `json:"items,omitempty"`
	Branch      *string `json:"branch,omitempty"`
}
type WorkflowRunTrigger struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Input       any    `json:"input,omitempty"`
	TriggeredAt string `json:"triggered_at"`
}
type WorkflowRunDocument struct {
	RunID             string                          `json:"run_id"`
	WorkflowID        string                          `json:"workflow_id"`
	WorkflowVersionID string                          `json:"workflow_version_id"`
	Status            string                          `json:"status"`
	Trigger           WorkflowRunTrigger              `json:"trigger"`
	Executions        map[string]WorkflowRunExecution `json:"executions"`
	Output            any                             `json:"output,omitempty"`
	Error             *string                         `json:"error,omitempty"`
}
type WorkflowRunResponse struct {
	Run            WorkflowRun          `json:"run"`
	Document       *WorkflowRunDocument `json:"document"`
	DefinitionText string               `json:"definitionText"`
}
type WorkflowConnectorEvent struct {
	Type          string         `json:"type"`
	Label         string         `json:"label"`
	Description   string         `json:"description,omitempty"`
	PayloadSchema map[string]any `json:"payloadSchema,omitempty"`
	OptionsSchema map[string]any `json:"optionsSchema,omitempty"`
}
type WorkflowConnectorProvider struct {
	ConnectorType   string                   `json:"connectorType"`
	Label           string                   `json:"label"`
	ResourceSubtype string                   `json:"resourceSubtype"`
	AppSlug         string                   `json:"appSlug,omitempty"`
	Events          []WorkflowConnectorEvent `json:"events"`
}

func (c *Client) ListWorkflowRuns(workflowID string) (*WorkflowRunListResponse, error) {
	var resp WorkflowRunListResponse
	err := c.doRequest("GET", "/workflows/"+url.PathEscape(workflowID)+"/runs", nil, &resp)
	return &resp, err
}
func (c *Client) GetWorkflowRun(workflowID, runID string) (*WorkflowRunResponse, error) {
	var resp WorkflowRunResponse
	err := c.doRequest("GET", "/workflows/"+url.PathEscape(workflowID)+"/runs/"+url.PathEscape(runID), nil, &resp)
	return &resp, err
}
func (c *Client) GetWorkflowConnectorEvents(connectorType string) (*WorkflowConnectorProvider, error) {
	var resp WorkflowConnectorProvider
	err := c.doRequest("GET", "/workflows/connector-events/"+url.PathEscape(connectorType), nil, &resp)
	return &resp, err
}
