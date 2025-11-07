package api

// --- Authentication / User structs ---

// LoginStartResponse represents the response from POST /login/start
type LoginStartResponse struct {
	Error           *AppErrorDetail `json:"error,omitempty"`
	DeviceCode      string          `json:"device_code,omitempty"`
	UserCode        string          `json:"user_code,omitempty"`
	VerificationURI string          `json:"verification_uri,omitempty"`
	ExpiresIn       int             `json:"expires_in,omitempty"`
	Interval        int             `json:"interval,omitempty"`
}

// LoginPollRequest represents the request body for POST /login/poll
type LoginPollRequest struct {
	DeviceCode string `json:"device_code"`
}

// LoginPollResponse represents the response from POST /login/poll
type LoginPollResponse struct {
	Error       *AppErrorDetail `json:"error,omitempty"`
	AccessToken string          `json:"access_token,omitempty"`
	TokenType   string          `json:"token_type,omitempty"`
	ExpiresIn   int             `json:"expires_in,omitempty"`
}

// VerifyTokenResponse represents the response from GET /verify
type VerifyTokenResponse struct {
	Error  *AppErrorDetail `json:"error,omitempty"`
	Active bool            `json:"active,omitempty"`
	UserID string          `json:"user_id,omitempty"`
	Email  string          `json:"email,omitempty"`
	Exp    int64           `json:"exp,omitempty"`
}

// --- Organization structs ---

// Organization represents an organization from the API
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrganizationsResponse represents the response from GET /organizations
type OrganizationsResponse struct {
	Error         *AppErrorDetail `json:"error,omitempty"`
	Organizations []Organization  `json:"organizations,omitempty"`
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
	Error          *AppErrorDetail `json:"error,omitempty"`
	ApplicationID  string          `json:"applicationId,omitempty"`
	RepositoryName string          `json:"repositoryName,omitempty"`
	CloneURLSSH    string          `json:"cloneUrlSsh,omitempty"`
	CloneURLHTTPS  string          `json:"cloneUrlHttps,omitempty"`
}

// GetApplicationByRepoRequest represents the request body for GET /application/from-repo
type GetApplicationByRepoRequest struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

// GetApplicationByRepoResponse represents the response from GET /application/from-repo
type GetApplicationByRepoResponse struct {
	Error         *AppErrorDetail `json:"error,omitempty"`
	ApplicationID string          `json:"applicationId,omitempty"`
}

// GetApplicationEnvRequest represents the request body for POST /application/env
type GetApplicationEnvRequest struct {
	OrganizationID string `json:"organizationId"`
	ApplicationID  string `json:"applicationId"`
}

// GetApplicationEnvResponse represents the response from POST /application/env
type GetApplicationEnvResponse struct {
	Error   *AppErrorDetail   `json:"error,omitempty"`
	EnvVars map[string]string `json:"envVars,omitempty"`
}

// ResourceItem represents a single resource
type ResourceItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetApplicationResourcesResponse represents the response from GET /applications/:applicationId/resources
type GetApplicationResourcesResponse struct {
	Error     *AppErrorDetail `json:"error,omitempty"`
	Resources []ResourceItem  `json:"resources,omitempty"`
}

// CreateApplicationVersionRequest represents the request body for POST /applications/versions
type CreateApplicationVersionRequest struct {
	ApplicationID string `json:"applicationId"`
}

// CreateApplicationVersionResponse represents the response from POST /applications/versions
type CreateApplicationVersionResponse struct {
	Error     *AppErrorDetail `json:"error,omitempty"`
	VersionID string          `json:"versionId,omitempty"`
}

// ApplicationItem represents a single application in the list
type ApplicationItem struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	GithubRepositoryName string `json:"githubRepositoryName"`
	CloneURLSSH          string `json:"cloneUrlSsh"`
	CloneURLHTTPS        string `json:"cloneUrlHttps"`
}

// GetOrganizationApplicationsRequest represents the request body for POST /organizations/applications
type GetOrganizationApplicationsRequest struct {
	OrganizationID string `json:"organizationId"`
}

// GetOrganizationApplicationsResponse represents the response from POST /organizations/applications
type GetOrganizationApplicationsResponse struct {
	Error        *AppErrorDetail   `json:"error,omitempty"`
	Applications []ApplicationItem `json:"applications,omitempty"`
}

// AddGithubCollaboratorsRequest represents the request body for POST /applications/add-gh-collaborators
type AddGithubCollaboratorsRequest struct {
	ApplicationID  string `json:"applicationId"`
	GithubUsername string `json:"githubUsername"`
}

// AddGithubCollaboratorsResponse represents the response from POST /applications/add-gh-collaborators
type AddGithubCollaboratorsResponse struct {
	Error   *AppErrorDetail `json:"error,omitempty"`
	Success bool            `json:"success,omitempty"`
	Message string          `json:"message,omitempty"`
}

// --- Version Check structs ---

// CheckVersionResponse represents the response from GET /version/check
type CheckVersionResponse struct {
	Error         *AppErrorDetail `json:"error,omitempty"`
	ForceUpgrade  bool            `json:"forceUpgrade,omitempty"`
	CanUpgrade    bool            `json:"canUpgrade,omitempty"`
	LatestVersion *string         `json:"latestVersion,omitempty"`
}

type VersionCheckRequest struct {
	Version string `json:"version"`
}
