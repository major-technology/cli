package api

import (
	"fmt"
	"os"
)

// AppEnvSetupRequest asks a human to configure app secret keys, never carries values.
type AppEnvSetupRequest struct {
	Kind         string   `json:"kind"`
	Keys         []string `json:"keys,omitempty"`
	ChatThreadID string   `json:"chatThreadId,omitempty"`
}

type AppEnvSetupResponse struct {
	Delivery string `json:"delivery"`
	Message  string `json:"message"`
	URL      string `json:"url,omitempty"`
}

func (c *Client) RequestAppEnvSetup(applicationID string, request AppEnvSetupRequest) (*AppEnvSetupResponse, error) {
	// Routing hint only; the server validates it against the sandbox token.
	request.ChatThreadID = os.Getenv("MAJOR_CHAT_THREAD_ID")
	var response AppEnvSetupResponse
	if err := c.doRequest("POST", fmt.Sprintf("/application/%s/env-variables/request-setup", applicationID), request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
