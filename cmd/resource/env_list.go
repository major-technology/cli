package resource

import (
	"encoding/json"
	"fmt"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagEnvListJSON bool

var envListCmd = &cobra.Command{
	Use:   "env-list",
	Short: "List available environments for this application",
	Long:  `List all available environments and show which one is currently selected.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runEnvList(cobraCmd)
	},
}

func init() {
	envListCmd.Flags().BoolVar(&flagEnvListJSON, "json", false, "Output in JSON format")
}

func runEnvList(cobraCmd *cobra.Command) error {
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	apiClient := singletons.GetAPIClient()

	currentEnvResp, err := apiClient.GetApplicationEnvironment(appInfo.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to get current environment", err)
	}

	envListResp, err := apiClient.ListApplicationEnvironments(appInfo.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to list environments", err)
	}

	if flagEnvListJSON {
		type envJSON struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			IsCurrent bool   `json:"isCurrent"`
		}

		envs := make([]envJSON, len(envListResp.Environments))
		for i, env := range envListResp.Environments {
			isCurrent := currentEnvResp.EnvironmentID != nil && env.ID == *currentEnvResp.EnvironmentID
			envs[i] = envJSON{
				ID:        env.ID,
				Name:      env.Name,
				IsCurrent: isCurrent,
			}
		}

		data, err := json.Marshal(envs)
		if err != nil {
			return errors.WrapError("failed to marshal JSON", err)
		}
		fmt.Fprintln(cobraCmd.OutOrStdout(), string(data))
		return nil
	}

	// Human-readable output
	cobraCmd.Println("\nEnvironments:")
	cobraCmd.Println("-------------")
	for _, env := range envListResp.Environments {
		if currentEnvResp.EnvironmentID != nil && env.ID == *currentEnvResp.EnvironmentID {
			cobraCmd.Printf("• %s (current)\n", env.Name)
		} else {
			cobraCmd.Printf("• %s\n", env.Name)
		}
	}
	cobraCmd.Println()
	return nil
}
