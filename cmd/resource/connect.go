package resource

import (
	"fmt"
	"strings"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var environmentFlag string

var connectCmd = &cobra.Command{
	Use:   "connect <resourceId> [resourceId...]",
	Short: "Open the browser to connect your OAuth accounts for per-user resources",
	Long: `Opens the /connect page in your browser so you can authenticate with
per-user OAuth resources. Resource IDs can be found via the web UI connectors page
or from the list_resources MCP tool.

If --environment is not provided, the app's current environment is used
(resolved from the git remote in the current directory).`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConnect(cmd, args)
	},
}

func init() {
	connectCmd.Flags().StringVar(&environmentFlag, "environment", "", "Environment ID (defaults to app's current environment)")
}

func runConnect(cmd *cobra.Command, resourceIds []string) error {
	cfg := singletons.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not initialized")
	}

	envID := environmentFlag

	if envID == "" {
		// Resolve environment from app context
		appInfo, err := utils.GetApplicationInfo("")
		if err != nil {
			return &errors.CLIError{
				Title:      "Could not resolve environment",
				Suggestion: "Use --environment <envId> or run this command from an app directory with a git remote.",
				Err:        err,
			}
		}

		apiClient := singletons.GetAPIClient()

		envResp, err := apiClient.GetApplicationEnvironment(appInfo.ApplicationID)
		if err != nil {
			return errors.WrapError("failed to get application environment", err)
		}

		if envResp.EnvironmentID == nil {
			return &errors.CLIError{
				Title:      "No environment set",
				Suggestion: "Use --environment <envId> or run 'major resource env' to choose one.",
			}
		}

		envID = *envResp.EnvironmentID
	}

	// Build the connect URL
	resources := strings.Join(resourceIds, ",")
	connectURL := fmt.Sprintf("%s/connect?resources=%s&environmentId=%s", cfg.FrontendURI, resources, envID)

	if err := utils.OpenBrowser(connectURL); err != nil {
		cmd.Printf("Failed to open browser automatically. Please visit:\n%s\n", connectURL)
		return nil
	}

	cmd.Printf("Opening connect page in your browser:\n%s\n", connectURL)
	return nil
}
