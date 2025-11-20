/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package user

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from the major app",
	Long:  `Logout revokes your CLI token and removes it from local storage`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runLogout(cobraCmd)
	},
}

func runLogout(cobraCmd *cobra.Command) error {
	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Call the logout endpoint to revoke the token (token will be fetched automatically)
	err := apiClient.Logout()
	if err != nil {
		return errors.WrapError("failed to logout", err)
	}

	// Delete the token from local keyring
	if err := mjrToken.DeleteToken(); err != nil {
		return errors.WrapError("failed to delete local token", err)
	}

	// Delete the default organization from local keyring (if it exists)
	err = mjrToken.DeleteDefaultOrg()
	if err != nil {
		return errors.WrapError("failed to delete default organization", err)
	}

	cobraCmd.Println("Successfully logged out!")
	return nil
}
