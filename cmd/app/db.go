package app

import (
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagDBStatusJSON bool

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Inspect the app's managed database",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var dbStatusCmd = &cobra.Command{
	Use:   "status",
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
		if flagDBStatusJSON {
			return utils.WriteJSON(cmd, resp)
		}
		cmd.Println(dbStatusText(resp))
		return nil
	},
}

func init() {
	dbStatusCmd.Flags().BoolVar(&flagDBStatusJSON, "json", false, "Output in JSON format")
	dbCmd.AddCommand(dbStatusCmd)
}

func dbStatusText(r *api.ManagedDatabaseStatusResponse) string {
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
