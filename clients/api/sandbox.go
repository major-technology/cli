package api

// SandboxUploadURLResponse is a single-use URL that writes one file into the calling sandbox.
type SandboxUploadURLResponse struct {
	UploadURL string `json:"uploadUrl"`
	ExpiresAt string `json:"expiresAt"`
}

// CreateSandboxUploadURL mints an upload URL for the sandbox the pod token belongs to.
func (c *Client) CreateSandboxUploadURL(destPath string) (*SandboxUploadURLResponse, error) {
	var resp SandboxUploadURLResponse
	err := c.doRequest("POST", "/sandbox-uploads", map[string]string{"destPath": destPath}, &resp)
	return &resp, err
}
