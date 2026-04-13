package vars

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	flagListEnv        string
	flagListShowValues bool
	flagListJSON       bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List environment variables for the current environment",
	Long: `List all environment variables for the selected environment.

By default values are masked. Pass --show-values to reveal them, or --json
to emit machine-readable output with full values.

Example:
  major vars list --env staging`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd)
	},
}

func init() {
	listCmd.Flags().StringVar(&flagListEnv, "env", "", "Target environment name (defaults to your current environment)")
	listCmd.Flags().BoolVar(&flagListShowValues, "show-values", false, "Show full values instead of masking them")
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format with full values")
}

type listJSONEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type listJSONOutput struct {
	Environment string          `json:"environment"`
	Variables   []listJSONEntry `json:"variables"`
}

func runList(cmd *cobra.Command) error {
	appID, err := getAppID()
	if err != nil {
		return err
	}

	env, err := resolveEnvironment(appID, flagListEnv)
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.GetEnvVariables(appID)
	if err != nil {
		return errors.WrapError("failed to get env variables", err)
	}

	// Filter to rows that have a value for the target environment
	type row struct {
		Key   string
		Value string
	}
	var rows []row
	for _, v := range resp.EnvVariables {
		if value, ok := findValueForEnv(v.Values, env.ID); ok {
			rows = append(rows, row{Key: v.Key, Value: value})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })

	if flagListJSON {
		out := listJSONOutput{
			Environment: env.Name,
			Variables:   make([]listJSONEntry, 0, len(rows)),
		}
		for _, r := range rows {
			out.Variables = append(out.Variables, listJSONEntry{Key: r.Key, Value: r.Value})
		}
		data, err := json.Marshal(out)
		if err != nil {
			return errors.WrapError("failed to encode JSON", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	cmd.Printf("Environment: %s\n\n", env.Name)

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
