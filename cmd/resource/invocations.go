package resource

import (
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagInvocationsEnvironment string
	flagInvocationsDays        int
	flagInvocationsSlow        bool
)

var invocationsCmd = &cobra.Command{
	Use:   "invocations",
	Short: "List an application's resource invocations, ranked by total time spent",
	Long: `Aggregates every resource call the app made over the window by operation
(the invocationKey for typed clients, METHOD host/path for proxy calls) and
reports calls, p50, p95, total time, error rate, and the call site when the
query extractor matched one. Use --slow to show only operations over 500 ms p95.
Sorted by total time descending.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appInfo, err := utils.GetApplicationInfo("")
		if err != nil {
			return errors.WrapError("failed to identify application", err)
		}

		resp, err := singletons.GetAPIClient().ListSlowOperations(appInfo.ApplicationID, api.ListSlowOperationsRequest{
			ExecutionEnvironment: flagInvocationsEnvironment,
			WindowDays:           flagInvocationsDays,
			IncludeAll:           !flagInvocationsSlow,
		})
		if err != nil {
			return errors.WrapError("failed to list resource invocations", err)
		}

		return utils.WriteJSON(cmd, resp)
	},
}

func init() {
	invocationsCmd.Flags().StringVar(&flagInvocationsEnvironment, "environment", "", "Traffic to include: deployment (default), coding-session, local-dev, or all")
	invocationsCmd.Flags().IntVar(&flagInvocationsDays, "days", 0, "Aggregation window in days, 1-30 (server default 7)")
	invocationsCmd.Flags().BoolVar(&flagInvocationsSlow, "slow", false, "Only include operations over 500 ms p95")
}
