package api

import (
	"encoding/json"
	"net/url"
)

// TargetVersion is the pull and push answer for every bundle kind. Each kind
// fills only its own id and display field, so the JSON output keeps the
// server's per-kind shape.
type TargetVersion struct {
	AgentID     string  `json:"agentId,omitempty"`
	SkillID     string  `json:"skillId,omitempty"`
	WorkflowID  string  `json:"workflowId,omitempty"`
	Name        string  `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Label       string  `json:"label,omitempty"`
	Version     int     `json:"version"`
	DownloadURL string  `json:"downloadUrl,omitempty"`
}

// DisplayName is the human name the kind carries: an agent's name, a skill's
// slug, or a workflow's label.
func (v TargetVersion) DisplayName() string {
	switch {
	case v.Name != "":
		return v.Name
	case v.Label != "":
		return v.Label
	case v.Slug != nil && *v.Slug != "":
		return *v.Slug
	case v.SkillID != "":
		return "skill " + v.SkillID
	default:
		return "bundle"
	}
}

type TargetUploadResponse struct {
	UploadURL string `json:"uploadUrl"`
	UploadKey string `json:"uploadKey"`
}
type TargetValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// TargetValidateResponse keeps warnings raw: agents and skills send strings,
// workflows send {path, message} objects.
type TargetValidateResponse struct {
	Valid    bool                    `json:"valid"`
	Errors   []TargetValidationError `json:"errors"`
	Warnings []json.RawMessage       `json:"warnings"`
}
type TriggerFailure struct {
	TriggerID     string   `json:"triggerId"`
	ConnectorType string   `json:"connectorType"`
	Events        []string `json:"events"`
	Error         string   `json:"error"`
}
type TargetPublishResponse struct {
	VersionID       string           `json:"versionId,omitempty"`
	WorkflowID      string           `json:"workflowId,omitempty"`
	Label           string           `json:"label,omitempty"`
	Version         int              `json:"version"`
	TriggerFailures []TriggerFailure `json:"triggerFailures,omitempty"`
}
type AgentFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// targetPath is /<apiPath>/<id>/<action>, e.g. /skills/<id>/pull.
func targetPath(apiPath, id, action string) string {
	return "/" + apiPath + "/" + url.PathEscape(id) + "/" + action
}
func (c *Client) PullTarget(apiPath, id string) (*TargetVersion, error) {
	var resp TargetVersion
	err := c.doRequest("POST", targetPath(apiPath, id, "pull"), map[string]any{}, &resp)
	return &resp, err
}
func (c *Client) TargetUploadURL(apiPath, id string) (*TargetUploadResponse, error) {
	var resp TargetUploadResponse
	err := c.doRequest("POST", targetPath(apiPath, id, "push-upload-url"), map[string]any{}, &resp)
	return &resp, err
}

// PushTarget saves an uploaded bundle. Notes are sent only when non-empty, so a
// kind that stores none (workflows) never receives them.
func (c *Client) PushTarget(apiPath, id, key, notes string) (*TargetVersion, error) {
	body := map[string]string{"uploadKey": key}
	if notes != "" {
		body["notes"] = notes
	}
	var resp TargetVersion
	err := c.doRequest("POST", targetPath(apiPath, id, "push"), body, &resp)
	return &resp, err
}

// ValidateTarget sends the kind's own validate body: {files} for agents,
// {uploadKey} for skills, {definitionText} for workflows.
func (c *Client) ValidateTarget(apiPath, id string, body map[string]any) (*TargetValidateResponse, error) {
	var resp TargetValidateResponse
	err := c.doRequest("POST", targetPath(apiPath, id, "validate"), body, &resp)
	return &resp, err
}
func (c *Client) PublishTarget(apiPath, id string) (*TargetPublishResponse, error) {
	var resp TargetPublishResponse
	err := c.doRequest("POST", targetPath(apiPath, id, "publish"), map[string]any{}, &resp)
	return &resp, err
}
