/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	mjrToken "github.com/major-technology/cli/token"
)

// LoginStartResponse represents the response from POST /cli/login/start
type LoginStartResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// LoginPollRequest represents the request body for POST /cli/login/poll
type LoginPollRequest struct {
	DeviceCode string `json:"device_code"`
}

// LoginPollResponse represents the response from POST /cli/login/poll
type LoginPollResponse struct {
	// Pending state
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`

	// Success state
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the major app",
	Long:  `Login and stores your session token`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runLogin(cmd))
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command) error {
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		return fmt.Errorf("api_url not configured")
	}

	startResp, err := startLogin(apiURL)
	cobra.CheckErr(err)

	if err := openBrowser(startResp.VerificationURI); err != nil {
		// ignore, failed to open browser
	}
	cmd.Println("Attempting to automatically open the SSO authorization page in your default browser.")
	cmd.Printf("If the browser does not open or you wish to use a different device to authorize this request, open the following URL:\n\n%s\n", startResp.VerificationURI)

	token, err := pollForToken(cmd, apiURL, startResp.DeviceCode, startResp.Interval, startResp.ExpiresIn)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := mjrToken.StoreToken(token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	cmd.Println("Successfully authenticated!")

	return nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// startLogin calls POST /cli/login/start
func startLogin(apiURL string) (*LoginStartResponse, error) {
	url := apiURL + "/login/start"

	resp, err := http.Post(url, "application/json", bytes.NewBufferString("{}"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s", errResp.Message)
		}
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var startResp LoginStartResponse
	if err := json.Unmarshal(body, &startResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	return &startResp, nil
}

// pollForToken polls POST /cli/login/poll until authenticated or timeout
func pollForToken(cmd *cobra.Command, apiURL string, deviceCode string, interval int, expiresIn int) (string, error) {
	url := apiURL + "/login/poll"

	pollReq := LoginPollRequest{
		DeviceCode: deviceCode,
	}

	reqBody, err := json.Marshal(pollReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	timeoutChan := time.After(time.Duration(expiresIn) * time.Second)

	for {
		select {
		case <-timeoutChan:
			return "", fmt.Errorf("authentication timeout - code expired")
		case <-ticker.C:
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
			if err != nil {
				return "", fmt.Errorf("failed to poll: %w", err)
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return "", fmt.Errorf("failed to read response: %w", err)
			}

			// Handle non-200 responses (expired/invalid codes)
			if resp.StatusCode == http.StatusBadRequest {
				var errResp ErrorResponse
				if err := json.Unmarshal(body, &errResp); err == nil {
					return "", fmt.Errorf("invalid device code: %s", errResp.Message)
				}
				return "", fmt.Errorf("bad request: %s", string(body))
			}

			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
			}

			var pollResp LoginPollResponse
			if err := json.Unmarshal(body, &pollResp); err != nil {
				return "", fmt.Errorf("failed to parse response: %w", err)
			}

			// Check if still pending
			if pollResp.Error == "authorization_pending" {
				cmd.Print(".")
				continue
			}

			// Success - got the token
			if pollResp.AccessToken != "" {
				cmd.Println() // New line after the dots
				return pollResp.AccessToken, nil
			}

			// Unexpected response
			return "", fmt.Errorf("unexpected response: %s", string(body))
		}
	}
}
