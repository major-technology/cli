package api

import "net/url"

// SkillItem is the web list's skill; the CLI decodes only what it prints.
type SkillItem struct {
	ID          string          `json:"id"`
	Slug        *string         `json:"slug"`
	Description *string         `json:"description"`
	Status      string          `json:"status"`
	Permissions ListPermissions `json:"permissions"`
}
type SkillListResponse struct {
	Skills []SkillItem `json:"skills"`
}
type SkillCreateResponse struct {
	SkillID string `json:"skillId"`
}

func (c *Client) ListSkills(organizationID string, includeReadOnly bool) (*SkillListResponse, error) {
	query := url.Values{"organizationId": {organizationID}}
	if includeReadOnly {
		query.Set("includeReadOnly", "true")
	}
	var resp SkillListResponse
	err := c.doRequest("GET", "/skills?"+query.Encode(), nil, &resp)
	return &resp, err
}
func (c *Client) CreateSkill(organizationID string) (*SkillCreateResponse, error) {
	var resp SkillCreateResponse
	err := c.doRequest("POST", "/skills", map[string]string{"organizationId": organizationID}, &resp)
	return &resp, err
}
