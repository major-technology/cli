package api

// --- Authentication / User structs ---

// LoginStartResponse represents the response from POST /login/start
type LoginStartResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// LoginPollRequest represents the request body for POST /login/poll
type LoginPollRequest struct {
	DeviceCode string `json:"device_code"`
}

// LoginPollResponse represents the response from POST /login/poll
type LoginPollResponse struct {
	// Pending state
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`

	// Success state
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

// VerifyTokenResponse represents the response from GET /verify
type VerifyTokenResponse struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

// --- Organization structs ---

// Organization represents an organization from the API
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrganizationsResponse represents the response from GET /organizations
type OrganizationsResponse struct {
	Organizations []Organization `json:"organizations"`
}

// --- Application structs ---

// CreateApplicationRequest represents the request body for POST /applications
type CreateApplicationRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	OrganizationID string `json:"organizationId"`
}

// CreateApplicationResponse represents the response from POST /applications
type CreateApplicationResponse struct {
	ApplicationID  string `json:"applicationId"`
	RepositoryName string `json:"repositoryName"`
	CloneURLSSH    string `json:"cloneUrlSsh"`
	CloneURLHTTPS  string `json:"cloneUrlHttps"`
}

// GetApplicationByRepoRequest represents the request body for GET /application/from-repo
type GetApplicationByRepoRequest struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

// GetApplicationByRepoResponse represents the response from GET /application/from-repo
type GetApplicationByRepoResponse struct {
	ApplicationID string `json:"applicationId"`
}

// GetApplicationEnvRequest represents the request body for POST /application/env
type GetApplicationEnvRequest struct {
	OrganizationID string `json:"organizationId"`
	ApplicationID  string `json:"applicationId"`
}

// GetApplicationEnvResponse represents the response from POST /application/env
type GetApplicationEnvResponse struct {
	EnvVars map[string]string `json:"envVars"`
}
