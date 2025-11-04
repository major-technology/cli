package org

import (
	"github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all organizations",
	Long:  `List all organizations you are a member of`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runList(cobraCmd))
	},
}

func runList(cobraCmd *cobra.Command) error {
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
