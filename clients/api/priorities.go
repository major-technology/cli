package api

import "net/url"

// PriorityItem is one priority in the list; the CLI decodes only what it uses.
type PriorityItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Version  int     `json:"version"`
	ClosedAt *string `json:"closedAt"`
}
type PriorityListResponse struct {
	Open   []PriorityItem `json:"open"`
	Closed []PriorityItem `json:"closed"`
}
type PriorityCreateResponse struct {
	PriorityID string `json:"priorityId"`
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
