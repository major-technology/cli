package org

import (
	"fmt"

	"github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/cmd/user"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// selectCmd represents the org select command
var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select a default organization",
	Long:  `Select a default organization from your available organizations.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runSelect(cobraCmd))
	},
}

func runSelect(cobraCmd *cobra.Command) error {
	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Fetch organizations (token will be fetched automatically)
	orgsResp, err := apiClient.GetOrganizations()
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return err
	}

	if len(orgsResp.Organizations) == 0 {
		cobraCmd.Println("No organizations available")
		return nil
	}

	// Let user select organization
	selectedOrg, err := user.SelectOrganization(cobraCmd, orgsResp.Organizations)
	if err != nil {
		return fmt.Errorf("failed to select organization: %w", err)
	}

	// Store the selected organization
	if err := mjrToken.StoreDefaultOrg(selectedOrg.ID, selectedOrg.Name); err != nil {
		return fmt.Errorf("failed to store default organization: %w", err)
	}

	cobraCmd.Printf("Default organization set to: %s\n", selectedOrg.Name)
	return nil
}
