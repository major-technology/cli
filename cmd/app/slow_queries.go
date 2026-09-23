package app

import (
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagSlowQueriesEnvironment string
	flagSlowQueriesDays        int
)

var slowQueriesCmd = &cobra.Command{
	Use:   "slow-queries",
	Short: "Inspect an application's slow resource operations",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var slowQueriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List resource operations ranked by total time spent, with p95 latency",
	Long: `Aggregates every resource call the app made over the window by operation
(the invocationKey for typed clients, METHOD host/path for proxy calls) and
reports calls, p50, p95, total time, error rate, and the call site when the
query extractor matched one. Sorted by total time descending.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		resp, err := singletons.GetAPIClient().ListSlowOperations(applicationID, api.ListSlowOperationsRequest{
			ExecutionEnvironment: flagSlowQueriesEnvironment,
			WindowDays:           flagSlowQueriesDays,
		})
		if err != nil {
			return errors.WrapError("failed to list slow operations", err)
		}

		return utils.WriteJSON(cmd, resp)
	},
}

func init() {
	slowQueriesListCmd.Flags().StringVar(&flagSlowQueriesEnvironment, "environment", "", "Traffic to include: deployment (default), coding-session, or local-dev")
	slowQueriesListCmd.Flags().IntVar(&flagSlowQueriesDays, "days", 0, "Aggregation window in days (1-30, default 7)")

	slowQueriesCmd.AddCommand(slowQueriesListCmd)
}
