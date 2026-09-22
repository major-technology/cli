package vars

import (
	"sort"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagListShowValues bool
	flagListJSON       bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List environment variables",
	Long: `List all environment variables for the current application.

By default values are masked. Pass --show-values to reveal them, or --json
to emit machine-readable output with full values.

Example:
  major vars list --show-values`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd)
	},
}

func init() {
	listCmd.Flags().BoolVar(&flagListShowValues, "show-values", false, "Show full values instead of masking them")
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format with full values")
}

type listJSONEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type listJSONOutput struct {
	Variables []listJSONEntry `json:"variables"`
}

func runList(cmd *cobra.Command) error {
	appID, err := getAppID()
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.GetEnvVariables(appID)
	if err != nil {
		return errors.WrapError("failed to get env variables", err)
	}

	type row struct {
		Key   string
		Value string
	}
	var rows []row
	for _, v := range resp.EnvVariables {
		rows = append(rows, row{Key: v.Key, Value: v.Value})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })

	if flagListJSON {
		out := listJSONOutput{
			Variables: make([]listJSONEntry, 0, len(rows)),
		}
		for _, r := range rows {
			out.Variables = append(out.Variables, listJSONEntry{Key: r.Key, Value: r.Value})
		}
		if err := utils.WriteJSON(cmd, out); err != nil {
			return errors.WrapError("failed to encode JSON", err)
		}
		return nil
	}

	if len(rows) == 0 {
		cmd.Println("No variables set.")
		return nil
	}

	// Compute column width for key
	maxKey := len("KEY")
	for _, r := range rows {
		if len(r.Key) > maxKey {
			maxKey = len(r.Key)
		}
	}

	cmd.Printf("%-*s  %s\n", maxKey, "KEY", "VALUE")
	for _, r := range rows {
		display := r.Value
		if !flagListShowValues {
			display = maskValue(r.Value)
		}
		cmd.Printf("%-*s  %s\n", maxKey, r.Key, display)
	}

	noun := "variables"
	if len(rows) == 1 {
		noun = "variable"
	}
	cmd.Printf("\n%d %s.\n", len(rows), noun)
	return nil
}
