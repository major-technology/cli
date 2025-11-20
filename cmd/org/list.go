package org

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all organizations",
	Long:  `List all organizations you are a member of`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runList(cobraCmd)
	},
}

func runList(cobraCmd *cobra.Command) error {
	apiClient := singletons.GetAPIClient()

	orgsResp, err := apiClient.GetOrganizations()
	if err != nil {
		return err
	}

	if len(orgsResp.Organizations) == 0 {
		return errors.ErrorNoOrganizationsAvailable
	}

	// Get the default org to mark it
	defaultOrgID, _, _ := mjrToken.GetDefaultOrg()

	// Print header
	cobraCmd.Println("\nYour Organizations:")
	cobraCmd.Println("-------------------")

	// Print each organization
	for _, org := range orgsResp.Organizations {
		if org.ID == defaultOrgID {
			cobraCmd.Printf("• %s (default)\n", org.Name)
		} else {
			cobraCmd.Printf("• %s\n", org.Name)
		}
	}

	cobraCmd.Println()
	return nil
}
