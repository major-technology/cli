package app

import (
	"fmt"
	"slices"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagXserverListJSON   bool
	flagXserverAddID      string
	flagXserverAddJSON    bool
	flagXserverRemoveID   string
	flagXserverRemoveJSON bool
)

var xserverCmd = &cobra.Command{
	Use:   "xserver",
	Short: "Manage the other apps this app may call",
	Long:  `Manage the list of other apps in this organization that the current app may call. The deploy grants access from the list.`,
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var xserverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the apps this app may call",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		appInfo, err := utils.GetApplicationInfo("")
		if err != nil {
			return errors.WrapError("failed to identify application", err)
		}

		resp, err := singletons.GetAPIClient().GetCallableApps(appInfo.ApplicationID)
		if err != nil {
			return errors.WrapError("failed to list callable apps", err)
		}

		items := resp.Applications

		if flagXserverListJSON {
			return utils.WriteJSON(cmd, items)
		}

		if len(items) == 0 {
			cmd.Println("This app does not call any other apps. Run 'major app xserver add --id <appId>' to add one.")
			return nil
		}

		for _, item := range items {
			line := fmt.Sprintf("%s  %s", item.ID, item.Name)
			if item.AppURL != nil {
				line += "  " + *item.AppURL
			}
			cmd.Println(line)
		}

		return nil
	},
}

var xserverAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Allow this app to call another app",
	Long:  `Add another application in this organization to the apps this app may call. Use 'major app list' to find application IDs. The change applies on the next deploy.`,
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		appInfo, err := utils.GetApplicationInfo("")
		if err != nil {
			return errors.WrapError("failed to identify application", err)
		}

		if flagXserverAddID == appInfo.ApplicationID {
			return fmt.Errorf("an application cannot call itself")
		}

		apiClient := singletons.GetAPIClient()

		// The server rejects an app from another org. The target is not read here,
		// because a sandbox token may view only its own app.
		current, err := apiClient.GetCallableApps(appInfo.ApplicationID)
		if err != nil {
			return errors.WrapError("failed to list callable apps", err)
		}

		ids := current.ApplicationIDs
		if !slices.Contains(ids, flagXserverAddID) {
			ids = append(ids, flagXserverAddID)
			if _, err := apiClient.SetCallableApps(appInfo.ApplicationID, ids); err != nil {
				return errors.WrapError("failed to update callable apps", err)
			}
		}

		if flagXserverAddJSON {
			return utils.WriteJSON(cmd, map[string]any{"appId": flagXserverAddID, "added": true})
		}

		cmd.Printf("This app may now call %s. The change applies on the next deploy.\n", flagXserverAddID)
		return nil
	},
}

var xserverRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Stop allowing this app to call another app",
	Long:  `Remove an application from the apps this app may call. The change applies on the next deploy.`,
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		appInfo, err := utils.GetApplicationInfo("")
		if err != nil {
			return errors.WrapError("failed to identify application", err)
		}

		apiClient := singletons.GetAPIClient()

		current, err := apiClient.GetCallableApps(appInfo.ApplicationID)
		if err != nil {
			return errors.WrapError("failed to list callable apps", err)
		}

		if !slices.Contains(current.ApplicationIDs, flagXserverRemoveID) {
			return fmt.Errorf("application with ID %q is not in this app's callable apps", flagXserverRemoveID)
		}

		ids := slices.DeleteFunc(slices.Clone(current.ApplicationIDs), func(id string) bool { return id == flagXserverRemoveID })
		if _, err := apiClient.SetCallableApps(appInfo.ApplicationID, ids); err != nil {
			return errors.WrapError("failed to update callable apps", err)
		}

		if flagXserverRemoveJSON {
			return utils.WriteJSON(cmd, map[string]any{"appId": flagXserverRemoveID, "removed": true})
		}

		cmd.Printf("Removed %s from the apps this app may call. The change applies on the next deploy.\n", flagXserverRemoveID)
		return nil
	},
}

func init() {
	xserverListCmd.Flags().BoolVar(&flagXserverListJSON, "json", false, "Output in JSON format")
	xserverAddCmd.Flags().StringVar(&flagXserverAddID, "id", "", "Application ID to allow calling")
	xserverAddCmd.Flags().BoolVar(&flagXserverAddJSON, "json", false, "Output in JSON format")
	xserverAddCmd.MarkFlagRequired("id")
	xserverRemoveCmd.Flags().StringVar(&flagXserverRemoveID, "id", "", "Application ID to stop calling")
	xserverRemoveCmd.Flags().BoolVar(&flagXserverRemoveJSON, "json", false, "Output in JSON format")
	xserverRemoveCmd.MarkFlagRequired("id")

	xserverCmd.AddCommand(xserverListCmd)
	xserverCmd.AddCommand(xserverAddCmd)
	xserverCmd.AddCommand(xserverRemoveCmd)
}
