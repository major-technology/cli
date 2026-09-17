package api

import (
	"fmt"
	"os"
)

// SetupRequest asks a human to configure app secret keys, never carries values.
type SetupRequest struct {
	Kind         string   `json:"kind"`
	Keys         []string `json:"keys,omitempty"`
	ChatThreadID string   `json:"chatThreadId,omitempty"`
}

type SetupResponse struct {
	Delivery string `json:"delivery"`
	Message  string `json:"message"`
	URL      string `json:"url,omitempty"`
}

func (c *Client) RequestSetup(applicationID string, request SetupRequest) (*SetupResponse, error) {
	// Routing hint only; the server validates it against the sandbox token.
	request.ChatThreadID = os.Getenv("MAJOR_CHAT_THREAD_ID")
	var response SetupResponse
	if err := c.doRequest("POST", fmt.Sprintf("/application/%s/request-setup", applicationID), request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
