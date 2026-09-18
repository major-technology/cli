package app

import (
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var aiProxyCmd = &cobra.Command{
	Use:   "ai-proxy",
	Short: "Inspect and enable the app's AI proxy",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var aiProxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the AI proxy is enabled and this month's spend",
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		resp, err := singletons.GetAPIClient().GetAiProxyStatus(applicationID)
		if err != nil {
			return errors.WrapError("failed to get AI proxy status", err)
		}

		return utils.WriteJSON(cmd, resp)
	},
}

var aiProxyEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable the AI proxy for this app",
	Long: `Enables the AI proxy for this app.

The proxy starts with a default spending limit of $10 per month, charged to the
organization. Requires application:edit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		if err := singletons.GetAPIClient().EnableAiProxy(applicationID); err != nil {
			return errors.WrapError("failed to enable the AI proxy", err)
		}

		cmd.Println("AI proxy enabled with a $10/month spending limit")
		return nil
	},
}

func init() {
	aiProxyCmd.AddCommand(aiProxyStatusCmd)
	aiProxyCmd.AddCommand(aiProxyEnableCmd)
}
