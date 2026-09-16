package app

import (
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagErrorsEnvironment string
	flagErrorsLimit       int
	flagErrorsSince       string
	flagErrorsUntil       string
	flagErrorsFixed       bool
)

var errorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Inspect an application's runtime errors",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var errorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent runtime errors",
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		resp, err := singletons.GetAPIClient().ListAppErrors(applicationID, api.ListAppErrorsRequest{
			Environment: flagErrorsEnvironment,
			Limit:       flagErrorsLimit,
			Since:       flagErrorsSince,
			Until:       flagErrorsUntil,
			Fixed:       flagErrorsFixed,
		})
		if err != nil {
			return errors.WrapError("failed to list app errors", err)
		}

		return utils.WriteJSON(cmd, resp)
	},
}

var errorsGetCmd = &cobra.Command{
	Use:   "get <errorId>",
	Short: "Show one error's full detail, including its stack trace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		detail, err := singletons.GetAPIClient().GetAppError(applicationID, args[0])
		if err != nil {
			return errors.WrapError("failed to get app error", err)
		}

		return utils.WriteJSON(cmd, detail)
	},
}

var errorsResolveCmd = &cobra.Command{
	Use:   "resolve <errorId>",
	Short: "Mark an error fixed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		if err := singletons.GetAPIClient().ResolveAppError(applicationID, args[0]); err != nil {
			return errors.WrapError("failed to resolve app error", err)
		}

		cmd.Printf("Marked error %s fixed\n", args[0])
		return nil
	},
}

var errorsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Record that this app reports runtime errors",
	Long: `Marks the app as having runtime error reporting enabled.

Run this only after the Major error-reporter scaffolding is committed to the
repo. The flag drives the dashboard's "monitoring active" state. Idempotent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		if err := singletons.GetAPIClient().EnableAppErrors(applicationID); err != nil {
			return errors.WrapError("failed to enable error reporting", err)
		}

		cmd.Println("Error reporting enabled")
		return nil
	},
}

func init() {
	errorsListCmd.Flags().StringVar(&flagErrorsEnvironment, "environment", "", "Filter by environment: deployment, coding-session, or local-dev")
	errorsListCmd.Flags().IntVar(&flagErrorsLimit, "limit", 0, "Maximum errors to return (1-100, default 100)")
	errorsListCmd.Flags().StringVar(&flagErrorsSince, "since", "", "Earliest time window bound, as RFC3339")
	errorsListCmd.Flags().StringVar(&flagErrorsUntil, "until", "", "Latest time window bound, as RFC3339")
	errorsListCmd.Flags().BoolVar(&flagErrorsFixed, "fixed", false, "Return fixed-and-not-regressed errors instead of active ones")

	errorsCmd.AddCommand(errorsListCmd)
	errorsCmd.AddCommand(errorsGetCmd)
	errorsCmd.AddCommand(errorsResolveCmd)
	errorsCmd.AddCommand(errorsEnableCmd)
}
