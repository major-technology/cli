package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	flagLogsLimit     int
	flagLogsSearch    string
	flagLogsSince     string
	flagLogsUntil     string
	flagLogsNextToken string
	flagLogsJSON      bool
)

func init() {
	logsCmd.Flags().IntVar(&flagLogsLimit, "limit", 0, "Maximum number of log lines to return (1-5000, default 500)")
	logsCmd.Flags().StringVar(&flagLogsSearch, "search", "", "Filter log lines by substring (case-sensitive)")
	logsCmd.Flags().StringVar(&flagLogsSince, "since", "", "Show logs since a duration (e.g. 30m, 1h) or RFC3339 timestamp")
	logsCmd.Flags().StringVar(&flagLogsUntil, "until", "", "Show logs up until an RFC3339 timestamp")
	logsCmd.Flags().StringVar(&flagLogsNextToken, "next-token", "", "Pagination cursor from a previous response")
	logsCmd.Flags().BoolVar(&flagLogsJSON, "json", false, "Output in JSON format")
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Display application logs",
	Long: `Display logs for the application in the current directory.

Logs are returned newest-first. When there are more logs than the limit,
a pagination cursor is printed that can be passed back with --next-token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogs(cmd)
	},
}

func runLogs(cmd *cobra.Command) error {
	applicationID, err := getApplicationID()
	if err != nil {
		return err
	}

	since, err := parseSinceFlag(flagLogsSince)
	if err != nil {
		return errors.WrapError("invalid --since value", err)
	}

	until, err := parseRFC3339(flagLogsUntil)
	if err != nil {
		return errors.WrapError("invalid --until value", err)
	}

	req := api.GetApplicationLogsRequest{
		Limit:     flagLogsLimit,
		Search:    flagLogsSearch,
		NextToken: flagLogsNextToken,
		Since:     since,
		Until:     until,
	}

	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.GetApplicationLogs(applicationID, req)
	if err != nil {
		return err
	}

	if flagLogsJSON {
		data, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	for _, entry := range resp.Logs {
		fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", entry.Ts, entry.Log)
	}

	if resp.NextToken != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "\n# more logs available — rerun with --next-token %s\n", resp.NextToken)
	}

	return nil
}

// parseSinceFlag accepts either a Go duration (e.g. "30m", "1h") relative to
// now, or an RFC3339 timestamp. Returns an RFC3339 string suitable for the API.
func parseSinceFlag(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d).UTC().Format(time.RFC3339Nano), nil
	}

	return parseRFC3339(s)
}

// parseRFC3339 validates and normalizes an RFC3339 timestamp string.
func parseRFC3339(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return "", err
		}
	}

	return t.UTC().Format(time.RFC3339Nano), nil
}
