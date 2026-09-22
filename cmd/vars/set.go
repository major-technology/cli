package vars

import (
	"strings"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var flagSetJSON bool

var setCmd = &cobra.Command{
	Use:   "set <KEY>=<VALUE>",
	Short: "Create or update an environment variable",
	Long: `Create or update a single environment variable.

Values may contain '=' characters; only the first '=' in the argument is
treated as the separator.

Example:
  major vars set DATABASE_URL=postgres://localhost/mydb`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSet(cmd, args[0])
	},
}

func init() {
	setCmd.Flags().BoolVar(&flagSetJSON, "json", false, "Output in JSON format")
}

func runSet(cmd *cobra.Command, arg string) error {
	idx := strings.Index(arg, "=")
	if idx <= 0 {
		return &errors.CLIError{
			Title:      "Invalid argument",
			Suggestion: "Pass the variable as KEY=VALUE, e.g. major vars set DATABASE_URL=postgres://...",
		}
	}
	key := arg[:idx]
	value := arg[idx+1:]

	if err := validateKey(key); err != nil {
		return err
	}

	appID, err := getAppID()
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()
	if _, err := apiClient.SetEnvVariable(appID, key, value); err != nil {
		return errors.WrapError("failed to set env variable", err)
	}

	if flagSetJSON {
		return utils.WriteJSON(cmd, map[string]any{
			"key":     key,
			"updated": true,
		})
	}

	cmd.Printf("Set %s.\n", key)
	return nil
}
