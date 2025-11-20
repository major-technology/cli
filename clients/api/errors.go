package api

import (
	"fmt"

	clierrors "github.com/major-technology/cli/errors"
)

// Error code constants from @repo/errors
const (
	// Authentication & Authorization Errors (2000-2099)
	ErrorCodeUnauthorized         = 2000
	ErrorCodeInvalidToken         = 2001 // Used for expired tokens
	ErrorCodeInvalidUserCode      = 2002
	ErrorCodeTokenNotFound        = 2003
	ErrorCodeInvalidDeviceCode    = 2004
	ErrorCodeAuthorizationPending = 2005

	// Organization Errors (3000-3099)
	ErrorCodeOrganizationNotFound = 3000
	ErrorCodeNotOrgMember         = 3001
	ErrorCodeNoCreatePermission   = 3002

	// Application Errors (4000-4099)
	ErrorCodeApplicationNotFound = 4000
	ErrorCodeNoApplicationAccess = 4001
	ErrorCodeDuplicateAppName    = 4002

	// GitHub Integration Errors (5000-5099)
	ErrorCodeGitHubRepoNotFound          = 5000
	ErrorCodeGitHubRepoAccessDenied      = 5001
	ErrorCodeGitHubCollaboratorAddFailed = 5002
)

// AppErrorDetail represents the error detail from the API (new format)
type AppErrorDetail struct {
	InternalCode int    `json:"internal_code"`
	ErrorString  string `json:"error_string"`
	StatusCode   int    `json:"status_code"`
}

// ErrorResponse represents an error response from the API (new format only)
type ErrorResponse struct {
	Error *AppErrorDetail `json:"error,omitempty"`
}

// errorCodeToCLIError maps API error codes to CLIError instances
var errorCodeToCLIError = map[int]*clierrors.CLIError{
	// Authentication & Authorization Errors (2000-2099)
	ErrorCodeUnauthorized:         clierrors.ErrorUnauthorized,
	ErrorCodeInvalidToken:         clierrors.ErrorInvalidToken,
	ErrorCodeInvalidUserCode:      clierrors.ErrorInvalidUserCode,
	ErrorCodeTokenNotFound:        clierrors.ErrorTokenNotFound,
	ErrorCodeInvalidDeviceCode:    clierrors.ErrorInvalidDeviceCode,
	ErrorCodeAuthorizationPending: clierrors.ErrorAuthorizationPending,

	// Organization Errors (3000-3099)
	ErrorCodeOrganizationNotFound: clierrors.ErrorOrganizationNotFoundAPI,
	ErrorCodeNotOrgMember:         clierrors.ErrorNotOrgMember,
	ErrorCodeNoCreatePermission:   clierrors.ErrorNoCreatePermission,

	// Application Errors (4000-4099)
	ErrorCodeApplicationNotFound: clierrors.ErrorApplicationNotFoundAPI,
	ErrorCodeNoApplicationAccess: clierrors.ErrorNoApplicationAccess,
	ErrorCodeDuplicateAppName:    clierrors.ErrorDuplicateAppName,

	// GitHub Integration Errors (5000-5099)
	ErrorCodeGitHubRepoNotFound:          clierrors.ErrorGitHubRepoNotFound,
	ErrorCodeGitHubRepoAccessDenied:      clierrors.ErrorGitHubRepoAccessDenied,
	ErrorCodeGitHubCollaboratorAddFailed: clierrors.ErrorGitHubCollaboratorAddFailed,
}

// ToCLIError converts an APIError to a CLIError
// If a specific error code mapping exists, it returns that CLIError
// Otherwise, it creates a generic CLIError with the API error details
func ToCLIError(errResp *ErrorResponse) error {
	// Check if we have a specific mapping for this error code
	if cliErr, exists := errorCodeToCLIError[errResp.Error.InternalCode]; exists {
		return cliErr
	}

	// No specific mapping - create a generic CLIError with API details
	return &clierrors.CLIError{
		Title:      fmt.Sprintf("API Error (Code: %d)", errResp.Error.InternalCode),
		Suggestion: "Please try again or contact support if the issue persists.",
		Err:        fmt.Errorf("%s", errResp.Error.ErrorString),
	}
}
