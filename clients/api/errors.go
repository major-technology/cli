package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"message"`
}

// APIError represents an API error with status code and message
type APIError struct {
	StatusCode int
	Message    string
	ErrorType  string
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

// CheckErr checks for errors and prints appropriate messages using the command's output
// Returns true if no error (ok to continue), false if there was an error
func CheckErr(cmd *cobra.Command, err error) bool {
	if err == nil {
		return true
	}

	// Check if it's a no token error
	if IsNoToken(err) {
		cmd.Println("Error: Not logged in. Please run 'major user login' first.")
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
