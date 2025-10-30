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

	"github.com/charmbracelet/huh"
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

// Organization represents an organization from the API
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OrganizationsResponse struct {
	Organizations []Organization `json:"organizations"`
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

	// Fetch organizations
	orgsResp, err := fetchOrganizations(apiURL, token)
	if err != nil {
		return fmt.Errorf("failed to fetch organizations: %w", err)
	}

	// Let user select default organization
	if len(orgsResp.Organizations) > 0 {
		selectedOrg, err := selectOrganization(cmd, orgsResp.Organizations)
		if err != nil {
			return fmt.Errorf("failed to select organization: %w", err)
		}

		if err := mjrToken.StoreDefaultOrg(selectedOrg.ID, selectedOrg.Name); err != nil {
			return fmt.Errorf("failed to store default organization: %w", err)
		}

		cmd.Printf("Default organization set to: %s\n", selectedOrg.Name)
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

// fetchOrganizations calls GET /organizations to retrieve the user's organizations
func fetchOrganizations(apiURL string, token string) (*OrganizationsResponse, error) {
	url := apiURL + "/organizations"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
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

	var orgsResp *OrganizationsResponse
	if err := json.Unmarshal(body, &orgsResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	return orgsResp, nil
}

// selectOrganization prompts the user to select an organization from the list
func selectOrganization(cmd *cobra.Command, orgs []Organization) (*Organization, error) {
	if len(orgs) == 0 {
		return nil, fmt.Errorf("no organizations available")
	}

	// If only one organization, automatically select it
	if len(orgs) == 1 {
		cmd.Printf("Only one organization available. Automatically selecting it.\n")
		return &orgs[0], nil
	}

	// Create options for huh select
	options := make([]huh.Option[string], len(orgs))
	for i, org := range orgs {
		options[i] = huh.NewOption(org.Name, org.ID)
	}

	var selectedID string

	// Create and run the select form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a default organization").
				Options(options...).
				Value(&selectedID),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get selection: %w", err)
	}

	// Find the selected organization
	for i, org := range orgs {
		if org.ID == selectedID {
			return &orgs[i], nil
		}
	}

	return nil, fmt.Errorf("selected organization not found")
}
