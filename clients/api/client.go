package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/pkg/errors"
)

// Client represents an API client for making authenticated requests
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client with the provided base URL and optional token
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequestWithoutAuth is a helper method to make unauthenticated HTTP requests
func (c *Client) doRequestWithoutAuth(method, path string, body interface{}, response interface{}) error {
	return c.doRequestInternal(method, path, body, response, false)
}

// doRequest is a helper method to make HTTP requests with common error handling
// It automatically gets the token from the keyring for each request
func (c *Client) doRequest(method, path string, body interface{}, response interface{}) error {
	return c.doRequestInternal(method, path, body, response, true)
}

// doRequestInternal is the internal implementation for making HTTP requests
func (c *Client) doRequestInternal(method, path string, body interface{}, response interface{}, requireAuth bool) error {
	var token string
	if requireAuth {
		// Get token from keyring for this request
		t, err := mjrToken.GetToken()
		if err != nil {
			return &NoTokenError{OriginalError: err}
		}
		token = t
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return errors.Wrap(err, "failed to marshal request body")
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to make request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response")
	}

	// Handle error responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			// Prefer error_description if available, otherwise use message
			message := errResp.ErrorDescription
			if message == "" {
				message = errResp.Message
			}
			if message == "" {
				message = string(respBody)
			}
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    message,
				ErrorType:  errResp.Error,
			}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	// Parse successful response if a response struct is provided
	if response != nil {
		if err := json.Unmarshal(respBody, response); err != nil {
			return errors.Wrap(err, "failed to parse response")
		}
	}

	return nil
}

// --- Authentication / User endpoints ---

// StartLogin initiates the device flow login process
func (c *Client) StartLogin() (*LoginStartResponse, error) {
	var resp LoginStartResponse
	err := c.doRequestWithoutAuth("POST", "/login/start", map[string]interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PollLogin polls the login endpoint to check if the user has authorized the device
func (c *Client) PollLogin(deviceCode string) (*LoginPollResponse, error) {
	req := LoginPollRequest{DeviceCode: deviceCode}

	var resp LoginPollResponse
	err := c.doRequestWithoutAuth("POST", "/login/poll", req, &resp)
	if err != nil {
		// Check if it's a bad request (expired/invalid codes)
		if IsBadRequest(err) {
			return nil, fmt.Errorf("invalid or expired device code")
		}
		return nil, err
	}

	return &resp, nil
}

// VerifyToken verifies the current token and returns user information
func (c *Client) VerifyToken() (*VerifyTokenResponse, error) {
	var resp VerifyTokenResponse
	err := c.doRequest("GET", "/verify", nil, &resp)
	if err != nil {
		if IsUnauthorized(err) {
			return nil, fmt.Errorf("invalid or expired token - please login again")
		}
		return nil, err
	}

	if !resp.Active {
		return nil, fmt.Errorf("token is not active - please login again")
	}

	return &resp, nil
}

// Logout revokes the current token
func (c *Client) Logout() error {
	return c.doRequest("POST", "/logout", map[string]interface{}{}, nil)
}

// --- Organization endpoints ---

// GetOrganizations retrieves the list of organizations for the authenticated user
func (c *Client) GetOrganizations() (*OrganizationsResponse, error) {
	var resp OrganizationsResponse
	err := c.doRequest("GET", "/organizations", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Application endpoints ---

// CreateApplication creates a new application with a GitHub repository
func (c *Client) CreateApplication(name, description, organizationID string) (*CreateApplicationResponse, error) {
	req := CreateApplicationRequest{
		Name:           name,
		Description:    description,
		OrganizationID: organizationID,
	}

	var resp CreateApplicationResponse
	err := c.doRequest("POST", "/applications", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetApplicationByRepo retrieves an application by its repository owner and name
func (c *Client) GetApplicationByRepo(owner, repo string) (*GetApplicationByRepoResponse, error) {
	req := GetApplicationByRepoRequest{
		Owner: owner,
		Repo:  repo,
	}

	var resp GetApplicationByRepoResponse
	err := c.doRequest("POST", "/application/from-repo", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetApplicationEnv retrieves environment variables for an application
func (c *Client) GetApplicationEnv(organizationID, applicationID string) (map[string]string, error) {
	req := GetApplicationEnvRequest{
		OrganizationID: organizationID,
		ApplicationID:  applicationID,
	}

	var resp GetApplicationEnvResponse
	err := c.doRequest("POST", "/application/env", req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.EnvVars, nil
}

// GetApplicationResources retrieves resources for an application
func (c *Client) GetApplicationResources(applicationID string) (*GetApplicationResourcesResponse, error) {
	path := fmt.Sprintf("/applications/%s/resources", applicationID)

	var resp GetApplicationResourcesResponse
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateApplicationVersion creates a new version of an application
func (c *Client) CreateApplicationVersion(applicationID string) (*CreateApplicationVersionResponse, error) {
	req := CreateApplicationVersionRequest{
		ApplicationID: applicationID,
	}

	var resp CreateApplicationVersionResponse
	err := c.doRequest("POST", "/applications/versions", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetOrganizationApplications retrieves all applications for an organization
func (c *Client) GetOrganizationApplications(organizationID string) (*GetOrganizationApplicationsResponse, error) {
	path := fmt.Sprintf("/organizations/applications?organizationId=%s", organizationID)

	var resp GetOrganizationApplicationsResponse
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
