package org

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

var idCmd = &cobra.Command{
	Use:    "id",
	Short:  "Print the default organization ID",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runID(cmd)
	},
}

func runID(cmd *cobra.Command) error {
	orgID, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return err
	}

	cmd.Print(orgID)
	return nil
}
