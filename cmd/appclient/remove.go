package appclient

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagRemoveID   string
	flagRemoveJSON bool
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a client for another application",
	PreRunE: middleware.ChainParent(
		middleware.CheckLogin,
		middleware.CheckNodeInstalled,
		middleware.CheckNodeVersion("22.12"),
		middleware.CheckPnpmInstalled,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runRemove(cobraCmd)
	},
}

func init() {
	removeCmd.Flags().StringVar(&flagRemoveID, "id", "", "Application ID of the client to remove")
	removeCmd.Flags().BoolVar(&flagRemoveJSON, "json", false, "Output in JSON format")
	removeCmd.MarkFlagRequired("id")
}

func runRemove(cobraCmd *cobra.Command) error {
	data, err := os.ReadFile("apps.json")
	if err != nil {
		return errors.WrapError("failed to read apps.json", err)
	}

	var entries []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return errors.WrapError("failed to parse apps.json", err)
	}

	name := ""
	for _, e := range entries {
		if e.ID == flagRemoveID {
			name = e.Name
			break
		}
	}

	if name == "" {
		return fmt.Errorf("no client for application %q", flagRemoveID)
	}

	if err := runAppClientCLI(cobraCmd, ".", "remove", name); err != nil {
		return err
	}

	if flagRemoveJSON {
		return utils.WriteJSON(cobraCmd, map[string]any{"appId": flagRemoveID, "removed": true})
	}

	cobraCmd.Printf("Removed client for %s (%s)\n", name, flagRemoveID)
	return nil
}
