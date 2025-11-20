package org

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/cmd/user"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// selectCmd represents the org select command
var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select a default organization",
	Long:  `Select a default organization from your available organizations.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runSelect(cobraCmd)
	},
}

func runSelect(cobraCmd *cobra.Command) error {
	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Fetch organizations (token will be fetched automatically)
	orgsResp, err := apiClient.GetOrganizations()
	if err != nil {
		return err
	}

	if len(orgsResp.Organizations) == 0 {
		return errors.ErrorNoOrganizationsAvailable
	}

	// Let user select organization
	selectedOrg, err := user.SelectOrganization(cobraCmd, orgsResp.Organizations)
	if err != nil {
		return errors.WrapError("failed to select organization", err)
	}

	// Store the selected organization
	if err := mjrToken.StoreDefaultOrg(selectedOrg.ID, selectedOrg.Name); err != nil {
		return errors.WrapError("failed to store default organization", err)
	}

	cobraCmd.Printf("Default organization set to: %s\n", selectedOrg.Name)
	return nil
}
