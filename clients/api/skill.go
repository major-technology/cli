package api

import "net/url"

func (c *Client) ListSkills(editable, published bool) (Record, error) {
	q := url.Values{}
	if editable {
		q.Set("editable", "true")
	}
	if published {
		q.Set("published", "true")
	}
	p := "/skills"
	if len(q) > 0 {
		p += "?" + q.Encode()
	}
	return c.agentRequest("GET", p, nil)
}
func (c *Client) GetSkill(id string) (Record, error) {
	return c.agentRequest("GET", "/skills/"+url.PathEscape(id), nil)
}
func (c *Client) CreateSkill() (Record, error) { return c.agentRequest("POST", "/skills", Record{}) }
func (c *Client) ListApps(editable bool) (Record, error) {
	p := "/apps"
	if editable {
		p += "?editable=true"
	}
	return c.agentRequest("GET", p, nil)
}
