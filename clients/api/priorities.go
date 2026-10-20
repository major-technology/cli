package api

import "net/url"

// PriorityItem is one priority in the list; the CLI decodes only what it uses.
type PriorityItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Position int     `json:"position"`
	Version  int     `json:"version"`
	ClosedAt *string `json:"closedAt"`
}
type PriorityListResponse struct {
	Open   []PriorityItem `json:"open"`
	Closed []PriorityItem `json:"closed"`
}
type PriorityCreateResponse struct {
	FileID string `json:"fileId"`
}

func (c *Client) ListPriorities(organizationID string) (*PriorityListResponse, error) {
	var resp PriorityListResponse
	err := c.doRequest("GET", "/priorities?"+url.Values{"organizationId": {organizationID}}.Encode(), nil, &resp)
	return &resp, err
}
func (c *Client) CreatePriority(organizationID, title string) (*PriorityCreateResponse, error) {
	var resp PriorityCreateResponse
	err := c.doRequest("POST", "/priorities", map[string]string{"organizationId": organizationID, "title": title}, &resp)
	return &resp, err
}
func (c *Client) DeletePriority(id string) error {
	return c.doRequest("DELETE", "/priorities/"+url.PathEscape(id), nil, nil)
}

// ValidatePriority sends the folder's raw priority.json (nil when it has none) and its entry paths.
func (c *Client) ValidatePriority(id string, priorityJSON *string, entries []string) (*TargetValidateResponse, error) {
	body := map[string]any{"entries": entries}
	if priorityJSON != nil {
		body["priorityJson"] = *priorityJSON
	}
	var resp TargetValidateResponse
	err := c.doRequest("POST", targetPath("priorities", id, "validate"), body, &resp)
	return &resp, err
}

// UpdatePriority sends the parsed priority.json as written, so the server's strict schema sees every field.
func (c *Client) UpdatePriority(id string, priority map[string]any) error {
	return c.doRequest("POST", targetPath("priorities", id, "update"), priority, nil)
}
