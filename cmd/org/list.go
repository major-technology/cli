package org

import (
	"encoding/json"
	"fmt"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var flagListJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all organizations",
	Long:  `List all organizations you are a member of`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runList(cobraCmd)
	},
}

func init() {
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format")
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

	if flagListJSON {
		type orgJSON struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			IsSelected bool   `json:"isSelected"`
		}

		orgs := make([]orgJSON, len(orgsResp.Organizations))
		for i, org := range orgsResp.Organizations {
			orgs[i] = orgJSON{
				ID:         org.ID,
				Name:       org.Name,
				IsSelected: org.ID == defaultOrgID,
			}
		}

		data, err := json.Marshal(orgs)
		if err != nil {
			return errors.WrapError("failed to marshal JSON", err)
		}
		fmt.Fprintln(cobraCmd.OutOrStdout(), string(data))
		return nil
	}

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
