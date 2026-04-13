package vars

import (
	"encoding/json"
	"fmt"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	flagGetEnv  string
	flagGetJSON bool
)

var getCmd = &cobra.Command{
	Use:   "get <KEY>",
	Short: "Print a single environment variable's value",
	Long: `Print the value of a single environment variable for the selected environment.

The raw value is written to stdout, with no prefix, suitable for shell use:
  export DATABASE_URL=$(major vars get DATABASE_URL)

Exits non-zero if the key does not exist or has no value in the target environment.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, args[0])
	},
}

func init() {
	getCmd.Flags().StringVar(&flagGetEnv, "env", "", "Target environment name (defaults to your current environment)")
	getCmd.Flags().BoolVar(&flagGetJSON, "json", false, "Output in JSON format: {key, value, environment}")
}

type getJSONOutput struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Environment string `json:"environment"`
}

func runGet(cmd *cobra.Command, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	appID, err := getAppID()
	if err != nil {
		return err
	}

	env, err := resolveEnvironment(appID, flagGetEnv)
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.GetEnvVariables(appID)
	if err != nil {
		return errors.WrapError("failed to get env variables", err)
	}

	for _, v := range resp.EnvVariables {
		if v.Key != key {
			continue
		}
		value, ok := findValueForEnv(v.Values, env.ID)
		if !ok {
			return &errors.CLIError{
				Title: fmt.Sprintf("%s has no value in %q environment", key, env.Name),
			}
		}
		if flagGetJSON {
			data, err := json.Marshal(getJSONOutput{Key: key, Value: value, Environment: env.Name})
			if err != nil {
				return errors.WrapError("failed to encode JSON", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), value)
		return nil
	}

	return &errors.CLIError{
		Title: fmt.Sprintf("%s is not set", key),
	}
}
