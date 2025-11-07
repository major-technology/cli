package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Error code constants from @repo/errors
const (
	// Authentication & Authorization Errors (2000-2099)
	ErrorCodeUnauthorized         = 2000
	ErrorCodeInvalidToken         = 2001
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

// ErrorResponse represents an error response from the API
// Supports both new format (error object) and legacy format (error string)
type ErrorResponse struct {
	// New format
	Error *AppErrorDetail `json:"error,omitempty"`

	// Legacy format (for backward compatibility)
	ErrorString string `json:"error_description,omitempty"`
	Message     string `json:"message,omitempty"`
}

// APIError represents an API error with status code and message
type APIError struct {
	StatusCode   int
	InternalCode int // Internal error code from the API
	Message      string
	ErrorType    string
}

func (e *APIError) Error() string {
	if e.ErrorType != "" {
		return fmt.Sprintf("API error (status %d): %s - %s", e.StatusCode, e.ErrorType, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// NoTokenError represents an error when no token is available
type NoTokenError struct {
	OriginalError error
}

func (e *NoTokenError) Error() string {
	return fmt.Sprintf("not logged in: %v", e.OriginalError)
}

// ForceUpgradeError represents an error when the CLI version is too old and must be upgraded
type ForceUpgradeError struct {
	LatestVersion string
}

func (e *ForceUpgradeError) Error() string {
	return "CLI version is out of date and must be upgraded"
}

// IsUnauthorized checks if the error is an unauthorized error
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsBadRequest checks if the error is a bad request error
func IsBadRequest(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusBadRequest
	}
	return false
}

// IsNoToken checks if the error is a no token error
func IsNoToken(err error) bool {
	var noTokenErr *NoTokenError
	return errors.As(err, &noTokenErr)
}

// HasErrorCode checks if the error has a specific internal error code
func HasErrorCode(err error, code int) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.InternalCode == code
	}
	return false
}

// IsAuthorizationPending checks if the error is an authorization pending error
func IsAuthorizationPending(err error) bool {
	return HasErrorCode(err, ErrorCodeAuthorizationPending)
}

// IsInvalidDeviceCode checks if the error is an invalid device code error
func IsInvalidDeviceCode(err error) bool {
	return HasErrorCode(err, ErrorCodeInvalidDeviceCode)
}

// GetErrorCode returns the internal error code from an error, or 0 if not an APIError
func GetErrorCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.InternalCode
	}
	return 0
}

// IsForceUpgrade checks if the error is a force upgrade error
func IsForceUpgrade(err error) bool {
	var forceUpgradeErr *ForceUpgradeError
	return errors.As(err, &forceUpgradeErr)
}

// CheckErr checks for errors and prints appropriate messages using the command's output
// Returns true if no error (ok to continue), false if there was an error
func CheckErr(cmd *cobra.Command, err error) bool {
	if err == nil {
		return true
	}

	// Check if it's a force upgrade error
	if IsForceUpgrade(err) {
		// Create styled error message box
		errorStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5F87")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5F87"))

		commandStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#87D7FF"))

		message := fmt.Sprintf("Your CLI version is out of date and must be upgraded.\n\nRun:\n%s",
			commandStyle.Render("brew update && brew upgrade major"))

		cmd.Println(errorStyle.Render(message))
		return false
	}

	// Check if it's a no token error
	if IsNoToken(err) {
		// Create styled error message box
		errorStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5F87")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5F87"))

		commandStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#87D7FF"))

		message := fmt.Sprintf("Not logged in!\n\nRun %s to get started.",
			commandStyle.Render("major user login"))

		cmd.Println(errorStyle.Render(message))
		return false
	}

	// Check if it's an API error
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		// Just print the error description/message, nothing else
		cmd.Printf("Error: %s\n", apiErr.Message)
		return false
	}

	// Generic error
	cmd.Printf("Error: %v\n", err)
	return false
}
