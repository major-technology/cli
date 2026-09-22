package vars

import (
	"fmt"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagGetJSON bool

var getCmd = &cobra.Command{
	Use:   "get <KEY>",
	Short: "Print a single environment variable's value",
	Long: `Print the value of a single environment variable.

The raw value is written to stdout, with no prefix, suitable for shell use:
  export DATABASE_URL=$(major vars get DATABASE_URL)

Exits non-zero if the key does not exist.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, args[0])
	},
}

func init() {
	getCmd.Flags().BoolVar(&flagGetJSON, "json", false, "Output in JSON format: {key, value}")
}

type getJSONOutput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func runGet(cmd *cobra.Command, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	appID, err := getAppID()
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
		if flagGetJSON {
			return utils.WriteJSON(cmd, getJSONOutput{Key: key, Value: v.Value})
		}
		fmt.Fprintln(cmd.OutOrStdout(), v.Value)
		return nil
	}

	return &errors.CLIError{
		Title: fmt.Sprintf("%s is not set", key),
	}
}
