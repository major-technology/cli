package api

import "net/url"

// RelatedArtifact is one member of an artifact's family: the anchor itself or
// an app, agent, workflow, or skill within two hops of references.
type RelatedArtifact struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Accessible bool   `json:"accessible"`
	IsAnchor   bool   `json:"isAnchor"`
}
type RelatedArtifactsResponse struct {
	Artifacts []RelatedArtifact `json:"artifacts"`
}

func (c *Client) GetRelatedArtifacts(kind, id string) (*RelatedArtifactsResponse, error) {
	var resp RelatedArtifactsResponse
	err := c.doRequest("GET", "/artifact-family/"+url.PathEscape(kind)+"/"+url.PathEscape(id), nil, &resp)
	return &resp, err
}
