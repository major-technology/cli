/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package user

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current authenticated user",
	Long:  `Display information about the currently authenticated user by verifying the stored token.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runWhoami(cobraCmd)
	},
}

func runWhoami(cobraCmd *cobra.Command) error {
	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Call the /verify endpoint (token will be fetched automatically)
	verifyResp, err := apiClient.VerifyToken()
	if err != nil {
		return err
	}

	// Print the user email
	cobraCmd.Printf("Logged in as: %s\n", verifyResp.Email)

	// Try to get and display the default organization
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err == nil && orgID != "" && orgName != "" {
		cobraCmd.Printf("Default organization: %s (%s)\n", orgName, orgID)
	}

	return nil
}
