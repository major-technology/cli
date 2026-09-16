package app

import (
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Read and apply the organization's brand themes",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var themeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the organization's saved themes",
	RunE: func(cmd *cobra.Command, args []string) error {
		appInfo, err := utils.GetApplicationInfo("")
		if err != nil {
			return errors.WrapError("failed to identify application", err)
		}

		resp, err := singletons.GetAPIClient().ListThemes(appInfo.OrganizationID)
		if err != nil {
			return errors.WrapError("failed to list themes", err)
		}

		return utils.WriteJSON(cmd, resp)
	},
}

var themeGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the app's design system as guidance to follow before frontend work",
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		resp, err := singletons.GetAPIClient().GetApplicationTheme(applicationID)
		if err != nil {
			return errors.WrapError("failed to get app theme", err)
		}

		if resp.Theme == nil || *resp.Theme == "" {
			cmd.Println("This app has no theme configured.")
			return nil
		}

		cmd.Println(*resp.Theme)
		return nil
	},
}

var themeApplyCmd = &cobra.Command{
	Use:   "apply <themeId>",
	Short: "Apply one of the organization's themes to this app",
	Long: `Re-brands the app to one of the organization's saved themes.

This is the only correct way to change the app's appearance. Never hand-edit
colors, fonts or logo markup, which bypasses the theme pipeline.

Pins the theme on the app, then writes the generated files into this checkout:
app/theme.css, lib/theme.ts and components/ui/logo.tsx. Runs anywhere the repo
is checked out, including a plain local clone.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		if err := singletons.GetAPIClient().ApplyApplicationTheme(applicationID, args[0]); err != nil {
			return errors.WrapError("failed to apply theme", err)
		}

		// Pinning only records the id. Writing the files is the CLI's job, which is what
		// makes this work outside a sandbox. Same generator app create and app start use.
		if err := generateThemeFiles(""); err != nil {
			return errors.WrapError("theme pinned but files were not written", err)
		}

		cmd.Printf("Applied theme %s and wrote app/theme.css, lib/theme.ts and components/ui/logo.tsx\n", args[0])
		return nil
	},
}

func init() {
	themeCmd.AddCommand(themeListCmd)
	themeCmd.AddCommand(themeGetCmd)
	themeCmd.AddCommand(themeApplyCmd)
}
