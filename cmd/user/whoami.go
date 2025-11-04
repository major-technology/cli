/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package user

import (
	"github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current authenticated user",
	Long:  `Display information about the currently authenticated user by verifying the stored token.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runWhoami(cobraCmd))
	},
}

func runWhoami(cobraCmd *cobra.Command) error {
	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Call the /verify endpoint (token will be fetched automatically)
	verifyResp, err := apiClient.VerifyToken()
	if ok := api.CheckErr(cobraCmd, err); !ok {
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
