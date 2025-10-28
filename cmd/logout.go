/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	mjrToken "github.com/major-technology/cli/token"
)

// LogoutResponse represents the response from POST /cli/logout
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from the major app",
	Long:  `Logout revokes your CLI token and removes it from local storage`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runLogout(cmd))
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command) error {
	// Get API URL from config
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		return fmt.Errorf("api_url not configured")
	}

	// Get the token from keyring
	token, err := mjrToken.GetToken()
	if err != nil {
		return fmt.Errorf("not logged in or token not found: %w", err)
	}

	// Call the logout endpoint to revoke the token
	if err := revokeToken(apiURL, token); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// Delete the token from local keyring
	if err := mjrToken.DeleteToken(); err != nil {
		return fmt.Errorf("failed to delete local token: %w", err)
	}

	cmd.Println("Successfully logged out!")
	return nil
}

// revokeToken calls POST /cli/logout to revoke the token on the server
func revokeToken(apiURL string, token string) error {
	url := apiURL + "/logout"

	// Create the request
	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add the Authorization header with Bearer token
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusOK:
		var logoutResp LogoutResponse
		if err := json.Unmarshal(body, &logoutResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		return nil

	case http.StatusUnauthorized:
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("unauthorized: %s", errResp.Message)
		}
		return fmt.Errorf("unauthorized: %s", string(body))

	case http.StatusNotFound:
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("token not found: %s", errResp.Message)
		}
		return fmt.Errorf("token not found: %s", string(body))

	case http.StatusInternalServerError:
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s", errResp.Message)
		}
		return fmt.Errorf("server error: %s", string(body))

	default:
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
}
