/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	mjrToken "github.com/major-technology/cli/token"
)

// VerifyTokenResponse represents the response from GET /verify
type VerifyTokenResponse struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current authenticated user",
	Long:  `Display information about the currently authenticated user by verifying the stored token.`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runWhoami(cmd))
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command) error {
	// Get the API URL from config
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		return fmt.Errorf("api_url not configured")
	}

	// Get the stored token
	token, err := mjrToken.GetToken()
	if err != nil {
		cmd.Println("Not logged in")
		return nil
	}

	// Call the /verify endpoint
	verifyResp, err := verifyToken(apiURL, token)
	if err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}

	// Print the user email
	cmd.Printf("Logged in as: %s\n", verifyResp.Email)

	// Try to get and display the default organization
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err == nil && orgID != "" && orgName != "" {
		cmd.Printf("Default organization: %s (%s)\n", orgName, orgID)
	}

	return nil
}

// verifyToken calls GET /verify with the bearer token
func verifyToken(apiURL string, token string) (*VerifyTokenResponse, error) {
	url := apiURL + "/verify"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid or expired token - please login again")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s", errResp.Message)
		}
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var verifyResp VerifyTokenResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !verifyResp.Active {
		return nil, fmt.Errorf("token is not active - please login again")
	}

	return &verifyResp, nil
}
