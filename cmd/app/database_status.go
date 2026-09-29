package app

import (
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagDatabaseStatusJSON bool

var databaseStatusCmd = &cobra.Command{
	Use:   "database-status",
	Short: "Show the managed database status of the current application",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}
		resp, err := singletons.GetAPIClient().GetManagedDatabaseStatus(applicationID)
		if err != nil {
			return err
		}
		if flagDatabaseStatusJSON {
			return utils.WriteJSON(cmd, resp)
		}
		cmd.Println(databaseStatusText(resp))
		return nil
	},
}

func init() {
	databaseStatusCmd.Flags().BoolVar(&flagDatabaseStatusJSON, "json", false, "Output in JSON format")
}

func databaseStatusText(r *api.ManagedDatabaseStatusResponse) string {
	lines := []string{"Status: " + r.Status}
	if r.ResourceID != nil && *r.ResourceID != "" {
		lines = append(lines, "Resource ID: "+*r.ResourceID)
	}
	if r.DatabaseID != nil && *r.DatabaseID != "" {
		lines = append(lines, "Database ID: "+*r.DatabaseID)
	}
	if r.Message != "" {
		lines = append(lines, r.Message)
	}
	return strings.Join(lines, "\n")
}
