package vars

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	flagUnsetEnv             string
	flagUnsetAllEnvironments bool
	flagUnsetYes             bool
)

var unsetCmd = &cobra.Command{
	Use:   "unset <KEY>",
	Short: "Remove an environment variable",
	Long: `Remove an environment variable.

By default, removes only the value for the target environment, preserving
values set in other environments. Pass --all-environments to delete the key
across every environment.

Prompts for confirmation unless --yes is passed.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUnset(cmd, args[0])
	},
}

func init() {
	unsetCmd.Flags().StringVar(&flagUnsetEnv, "env", "", "Target environment name (defaults to your current environment)")
	unsetCmd.Flags().BoolVar(&flagUnsetAllEnvironments, "all-environments", false, "Remove the key across every environment")
	unsetCmd.Flags().BoolVarP(&flagUnsetYes, "yes", "y", false, "Skip the confirmation prompt")
}

func runUnset(cmd *cobra.Command, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	appID, err := getAppID()
	if err != nil {
		return err
	}

	var envID, envName string
	if !flagUnsetAllEnvironments {
		env, err := resolveEnvironment(appID, flagUnsetEnv)
		if err != nil {
			return err
		}
		envID = env.ID
		envName = env.Name
	} else if flagUnsetEnv != "" {
		return &errors.CLIError{
			Title:      "Conflicting flags",
			Suggestion: "--env and --all-environments cannot be used together.",
		}
	}

	if !flagUnsetYes {
		var prompt string
		if flagUnsetAllEnvironments {
			prompt = fmt.Sprintf("Remove %s from ALL environments?", key)
		} else {
			prompt = fmt.Sprintf("Remove %s from %q environment?", key, envName)
		}
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title(prompt).Value(&confirm),
			),
		)
		if err := form.Run(); err != nil {
			return errors.WrapError("failed to read confirmation", err)
		}
		if !confirm {
			return errors.ErrorOperationCancelled
		}
	}

	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.DeleteEnvVariableByKey(appID, key, envID, flagUnsetAllEnvironments)
	if err != nil {
		return errors.WrapError("failed to unset env variable", err)
	}

	switch {
	case !resp.Deleted && flagUnsetAllEnvironments:
		cmd.Printf("%s was not set.\n", key)
	case !resp.Deleted:
		cmd.Printf("Environment: %s\n", envName)
		cmd.Printf("%s was not set in %q.\n", key, envName)
	case flagUnsetAllEnvironments:
		cmd.Printf("Removed %s from all environments.\n", key)
	default:
		cmd.Printf("Environment: %s\n", envName)
		if resp.RemovedRow {
			cmd.Printf("Removed %s (last value, key deleted).\n", key)
		} else {
			cmd.Printf("Removed %s.\n", key)
		}
	}
	return nil
}
