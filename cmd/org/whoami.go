/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package org

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

// whoamiCmd represents the org whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current default organization",
	Long:  `Display information about the currently selected default organization.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runWhoami(cobraCmd)
	},
}

func runWhoami(cobraCmd *cobra.Command) error {
	// Try to get the default organization
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err != nil {
		cobraCmd.Println("No default organization set")
		return nil
	}

	if orgID == "" || orgName == "" {
		cobraCmd.Println("No default organization set")
		return nil
	}

	cobraCmd.Printf("Default organization: %s (%s)\n", orgName, orgID)
	return nil
}
