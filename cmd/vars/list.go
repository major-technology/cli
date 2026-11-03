package vars

import (
	"sort"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
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

By default values are masked and never fetched. Pass --show-values to reveal
them, or --json to emit machine-readable output with full values.

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
	info, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	env, err := resolveEnvironment(info.ApplicationID, flagListEnv)
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.GetEnvVariables(info.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to get env variables", err)
	}

	// Keys that have a value in the target environment. The listing carries no values;
	// they are fetched only when the caller asked to see them.
	var keys []string
	for _, v := range resp.EnvVariables {
		if isSetInEnv(v, env.ID) {
			keys = append(keys, v.Key)
		}
	}
	sort.Strings(keys)

	values := map[string]string{}
	if (flagListShowValues || flagListJSON) && len(keys) > 0 {
		values, err = fetchEnvValues(cmd, info, env, flagListEnv)
		if err != nil {
			return err
		}
	}

	if flagListJSON {
		out := listJSONOutput{
			Environment: env.Name,
			Variables:   make([]listJSONEntry, 0, len(keys)),
		}
		for _, k := range keys {
			out.Variables = append(out.Variables, listJSONEntry{Key: k, Value: values[k]})
		}
		if err := utils.WriteJSON(cmd, out); err != nil {
			return errors.WrapError("failed to encode JSON", err)
		}
		return nil
	}

	cmd.Printf("Environment: %s\n\n", env.Name)

	if len(keys) == 0 {
		cmd.Println("No variables set.")
		return nil
	}

	// Compute column width for key
	maxKey := len("KEY")
	for _, k := range keys {
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}

	cmd.Printf("%-*s  %s\n", maxKey, "KEY", "VALUE")
	for _, k := range keys {
		display := maskedValue
		if flagListShowValues {
			display = values[k]
		}
		cmd.Printf("%-*s  %s\n", maxKey, k, display)
	}

	noun := "variables"
	if len(keys) == 1 {
		noun = "variable"
	}
	cmd.Printf("\n%d %s.\n", len(keys), noun)
	return nil
}
