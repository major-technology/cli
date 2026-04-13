package mcp

import (
	"fmt"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

var getHeadersCmd = &cobra.Command{
	Use:    "get-headers",
	Short:  "Output MCP authentication headers as JSON",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := mjrToken.GetToken()
		if err != nil {
			return err
		}

		orgID, _, err := mjrToken.GetDefaultOrg()
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), `{"Authorization": "Bearer %s", "x-major-org-id": "%s"}`, token, orgID)
		return nil
	},
}
