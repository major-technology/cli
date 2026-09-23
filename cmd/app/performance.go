package app

import (
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagPerformanceEnvironment string
	flagPerformanceDays        int
	flagPerformanceAll         bool
)

var performanceCmd = &cobra.Command{
	Use:   "performance",
	Short: "Inspect an application's resource-call performance",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var performanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List resource operations over 500 ms p95, ranked by total time spent",
	Long: `Aggregates every resource call the app made over the window by operation
(the invocationKey for typed clients, METHOD host/path for proxy calls) and
reports calls, p50, p95, total time, error rate, and the call site when the
query extractor matched one. Only operations whose p95 exceeds 500 ms are
returned unless --all is set. Sorted by total time descending.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationID, err := getApplicationID()
		if err != nil {
			return err
		}

		resp, err := singletons.GetAPIClient().ListSlowOperations(applicationID, api.ListSlowOperationsRequest{
			ExecutionEnvironment: flagPerformanceEnvironment,
			WindowDays:           flagPerformanceDays,
			IncludeAll:           flagPerformanceAll,
		})
		if err != nil {
			return errors.WrapError("failed to list slow operations", err)
		}

		return utils.WriteJSON(cmd, resp)
	},
}

func init() {
	performanceListCmd.Flags().StringVar(&flagPerformanceEnvironment, "environment", "", "Traffic to include: deployment (default), coding-session, local-dev, or all")
	performanceListCmd.Flags().IntVar(&flagPerformanceDays, "days", 0, "Aggregation window in days, 1-30 (server default 7)")
	performanceListCmd.Flags().BoolVar(&flagPerformanceAll, "all", false, "Include operations at or under 500 ms p95")

	performanceCmd.AddCommand(performanceListCmd)
}
