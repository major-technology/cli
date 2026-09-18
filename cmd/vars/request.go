package vars

import (
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

func init() {
	var keys []string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "request --keys KEY[,KEY...]",
		Short: "Ask the user to enter secret values in the frontend",
		Args:  utils.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return runRequest(cmd, keys, jsonOutput) },
	}
	command.Flags().StringSliceVar(&keys, "keys", nil, "Environment variable keys to configure (no values)")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	command.MarkFlagRequired("keys")
	Cmd.AddCommand(command)
}

func runRequest(cmd *cobra.Command, keys []string, jsonOutput bool) error {
	for _, key := range keys {
		if err := validateKey(key); err != nil {
			return err
		}
	}
	appID, err := getAppID()
	if err != nil {
		return err
	}
	response, err := singletons.GetAPIClient().RequestAppEnvSetup(appID, api.AppEnvSetupRequest{Kind: "env", Keys: keys})
	if err != nil {
		return errors.WrapError("failed to request environment variable setup", err)
	}
	if jsonOutput {
		return utils.WriteJSON(cmd, response)
	}
	cmd.Println(response.Message)
	return nil
}
