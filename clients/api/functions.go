package api

import "net/url"

// FunctionItem is one function of a workflow. Version is the latest saved
// version and PublishedVersion the live one; each is null until it exists.
type FunctionItem struct {
	ID               string `json:"id"`
	Slug             string `json:"slug"`
	Version          *int   `json:"version"`
	PublishedVersion *int   `json:"publishedVersion"`
}
type FunctionListResponse struct {
	Functions []FunctionItem `json:"functions"`
}

// FunctionCreateResponse names the function a workflow has under a slug.
// Version is null when nothing has been pushed to it yet.
type FunctionCreateResponse struct {
	FunctionID string `json:"functionId"`
	Slug       string `json:"slug"`
	Version    *int   `json:"version"`
}

func (c *Client) ListFunctions(workflowID string) (*FunctionListResponse, error) {
	query := url.Values{"workflowId": {workflowID}}
	var resp FunctionListResponse
	err := c.doRequest("GET", "/functions?"+query.Encode(), nil, &resp)
	return &resp, err
}

// CreateFunction returns the workflow's function with this slug, creating it
// when the workflow has none.
func (c *Client) CreateFunction(workflowID, slug string) (*FunctionCreateResponse, error) {
	var resp FunctionCreateResponse
	err := c.doRequest("POST", "/functions", map[string]string{"workflowId": workflowID, "slug": slug}, &resp)
	return &resp, err
}
