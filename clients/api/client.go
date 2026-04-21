package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	mjrToken "github.com/major-technology/cli/clients/token"
	clierrors "github.com/major-technology/cli/errors"
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
			// User is not logged in - return appropriate CLIError
			return clierrors.ErrorNotLoggedIn
		}
		token = t
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return clierrors.WrapError("failed to marshal request body", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return clierrors.WrapError("failed to create request", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return clierrors.WrapError("failed to make request", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return clierrors.WrapError("failed to read response", err)
	}

	// Handle error responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp *ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != nil {
			return ToCLIError(errResp)
		}
		// Fallback for unexpected error format
		errResp = &ErrorResponse{
			Error: &AppErrorDetail{
				InternalCode: 9999,
				ErrorString:  string(respBody),
				StatusCode:   resp.StatusCode,
			},
		}
		return ToCLIError(errResp)
	}

	// Parse successful response if a response struct is provided
	if response != nil {
		if err := json.Unmarshal(respBody, response); err != nil {
			return clierrors.WrapError("failed to parse response", err)
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
		return nil, err
	}

	return &resp, nil
}

// VerifyToken verifies the current token and returns user information
func (c *Client) VerifyToken() (*VerifyTokenResponse, error) {
	var resp VerifyTokenResponse
	err := c.doRequest("GET", "/verify", nil, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Active {
		return nil, clierrors.ErrorTokenNotActive
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
func (c *Client) CreateApplication(name, description, organizationID string, themeID *string) (*CreateApplicationResponse, error) {
	req := CreateApplicationRequest{
		Name:           name,
		Description:    description,
		OrganizationID: organizationID,
		ThemeID:        themeID,
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
func (c *Client) CreateApplicationVersion(applicationID string, appURL string) (*CreateApplicationVersionResponse, error) {
	req := CreateApplicationVersionRequest{
		ApplicationID: applicationID,
		AppURL:        appURL,
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
	req := GetOrganizationApplicationsRequest{
		OrganizationID: organizationID,
	}

	var resp GetOrganizationApplicationsResponse
	err := c.doRequest("POST", "/organizations/applications", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// AddGithubCollaborators adds the user as a collaborator to the GitHub repository
func (c *Client) AddGithubCollaborators(applicationID, githubUsername string) (*AddGithubCollaboratorsResponse, error) {
	req := AddGithubCollaboratorsRequest{
		ApplicationID:  applicationID,
		GithubUsername: githubUsername,
	}

	var resp AddGithubCollaboratorsResponse
	err := c.doRequest("POST", "/applications/add-gh-collaborators", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetVersionStatus retrieves the deployment status of an application version
func (c *Client) GetVersionStatus(applicationID, organizationID, versionID string) (*GetVersionStatusResponse, error) {
	req := GetVersionStatusRequest{
		ApplicationID:  applicationID,
		OrganizationID: organizationID,
		VersionID:      versionID,
	}

	var resp GetVersionStatusResponse
	err := c.doRequest("POST", "/applications/versions/status", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Resource endpoints ---

// GetResources retrieves all resources for an organization
func (c *Client) GetResources(organizationID string) (*GetResourcesResponse, error) {
	req := GetResourcesRequest{
		OrganizationID: organizationID,
	}

	var resp GetResourcesResponse
	err := c.doRequest("POST", "/resources", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// SaveApplicationResources saves the selected resources for an application
func (c *Client) SaveApplicationResources(organizationID, applicationID string, resourceIDs []string) (*SaveApplicationResourcesResponse, error) {
	req := SaveApplicationResourcesRequest{
		OrganizationID: organizationID,
		ApplicationID:  applicationID,
		ResourceIDs:    resourceIDs,
	}

	var resp SaveApplicationResourcesResponse
	err := c.doRequest("POST", "/application-resources", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}


// --- Version Check endpoints ---

// CheckVersion checks if the CLI version is up to date
func (c *Client) CheckVersion(currentVersion string) (*CheckVersionResponse, error) {
	req := VersionCheckRequest{Version: currentVersion}

	var resp CheckVersionResponse
	err := c.doRequestWithoutAuth("POST", "/version/check", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Demo endpoints ---

// CreateDemoApplication creates a new demo application with a GitHub repository
func (c *Client) CreateDemoApplication(organizationID string) (*CreateDemoApplicationResponse, error) {
	req := CreateDemoApplicationRequest{
		OrganizationID: organizationID,
	}

	var resp CreateDemoApplicationResponse
	err := c.doRequest("POST", "/demo_application", req, &resp)
	if err != nil {
		fmt.Printf("Printing error: %+v\n", err)
		return nil, err
	}
	return &resp, nil
}

// GetDemoResource retrieves the singular demo resource
func (c *Client) GetDemoResource(orgID string) (*GetDemoResourceResponse, error) {
	var resp GetDemoResourceResponse
	path := fmt.Sprintf("/demo-resource?organizationId=%s", orgID)
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Environment endpoints ---

// GetApplicationEnvironment retrieves the user's current environment choice for an application
func (c *Client) GetApplicationEnvironment(applicationID string) (*GetApplicationEnvironmentResponse, error) {
	path := fmt.Sprintf("/application/%s/environment", applicationID)

	var resp GetApplicationEnvironmentResponse
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListApplicationEnvironments retrieves all available environments for an application
func (c *Client) ListApplicationEnvironments(applicationID string) (*ListEnvironmentsResponse, error) {
	path := fmt.Sprintf("/application/%s/environments", applicationID)

	var resp ListEnvironmentsResponse
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetApplicationEnvironment sets the user's environment choice for an application
func (c *Client) SetApplicationEnvironment(applicationID, environmentID string) (*SetEnvironmentChoiceResponse, error) {
	path := fmt.Sprintf("/application/%s/environment", applicationID)
	req := SetEnvironmentChoiceRequest{
		EnvironmentID: environmentID,
	}

	var resp SetEnvironmentChoiceResponse
	err := c.doRequest("POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetApplicationInfo retrieves application info including deploy status and URL
func (c *Client) GetApplicationInfo(applicationID string) (*GetApplicationInfoResponse, error) {
	var resp GetApplicationInfoResponse
	path := fmt.Sprintf("/applications/%s/info", applicationID)
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetApplicationForLink retrieves application info needed for the link command
func (c *Client) GetApplicationForLink(applicationID string) (*GetApplicationForLinkResponse, error) {
	var resp GetApplicationForLinkResponse
	path := fmt.Sprintf("/application/%s/link-info", applicationID)
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Env variable endpoints ---

// GetEnvVariables retrieves all env variables for an application, including per-environment values
func (c *Client) GetEnvVariables(applicationID string) (*GetEnvVariablesResponse, error) {
	path := fmt.Sprintf("/application/%s/env-variables", applicationID)
	var resp GetEnvVariablesResponse
	if err := c.doRequest("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetEnvVariable creates or updates a single env variable's value for a specific environment
func (c *Client) SetEnvVariable(applicationID, key, environmentID, value string) (*SetEnvVariableResponse, error) {
	path := fmt.Sprintf("/application/%s/env-variables/set", applicationID)
	req := SetEnvVariableRequest{
		Key:           key,
		EnvironmentID: environmentID,
		Value:         value,
	}
	var resp SetEnvVariableResponse
	if err := c.doRequest("POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteEnvVariableByKey deletes an env variable's value for a single environment, or the entire row.
// Pass environmentID for single-env removal (and allEnvironments=false), or allEnvironments=true
// (with empty environmentID) to remove the entire row across all environments.
func (c *Client) DeleteEnvVariableByKey(applicationID, key, environmentID string, allEnvironments bool) (*DeleteEnvVariableResponse, error) {
	path := fmt.Sprintf("/application/%s/env-variables/by-key/%s", applicationID, url.PathEscape(key))
	query := url.Values{}
	if allEnvironments {
		query.Set("allEnvironments", "true")
	} else if environmentID != "" {
		query.Set("environmentId", environmentID)
	}
	if encoded := query.Encode(); encoded != "" {
		path = path + "?" + encoded
	}
	var resp DeleteEnvVariableResponse
	if err := c.doRequest("DELETE", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Theme endpoints ---

// GetThemeFiles retrieves the generated theme files for an application
func (c *Client) GetThemeFiles(applicationID string) (*GetThemeFilesResponse, error) {
	path := fmt.Sprintf("/application/%s/theme-files", applicationID)
	var resp GetThemeFilesResponse
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListThemes retrieves all themes for an organization
func (c *Client) ListThemes(orgID string) (*ListThemesResponse, error) {
	path := fmt.Sprintf("/themes?organizationId=%s", orgID)
	var resp ListThemesResponse
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetThemeVersion retrieves the theme version for an application
func (c *Client) GetThemeVersion(applicationID string) (*GetThemeVersionResponse, error) {
	path := fmt.Sprintf("/application/%s/theme-version", applicationID)
	var resp GetThemeVersionResponse
	err := c.doRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpgradeTheme bumps the app's theme version to the latest
func (c *Client) UpgradeTheme(applicationID string) error {
	path := fmt.Sprintf("/application/%s/upgrade-theme", applicationID)
	var resp struct {
		Error *AppErrorDetail `json:"error,omitempty"`
	}
	return c.doRequest("POST", path, struct{}{}, &resp)
}

// --- Application log endpoints ---

// GetApplicationLogs retrieves paginated logs for an application.
func (c *Client) GetApplicationLogs(applicationID string, req GetApplicationLogsRequest) (*GetApplicationLogsResponse, error) {
	query := url.Values{}
	if req.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Search != "" {
		query.Set("search", req.Search)
	}
	if req.NextToken != "" {
		query.Set("nextToken", req.NextToken)
	}
	if req.Since != "" {
		query.Set("since", req.Since)
	}
	if req.Until != "" {
		query.Set("until", req.Until)
	}

	path := fmt.Sprintf("/applications/%s/logs", applicationID)
	if encoded := query.Encode(); encoded != "" {
		path = path + "?" + encoded
	}

	var resp GetApplicationLogsResponse
	if err := c.doRequest("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

