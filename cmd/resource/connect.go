package resource

import (
	"fmt"
	"strings"

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

If --environment is not provided, the environment is auto-resolved from
the app's git remote. If that also fails, the connect page will use
the org's default environment.`,
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

	// Build the connect URL
	resources := strings.Join(resourceIds, ",")
	connectURL := fmt.Sprintf("%s/connect?resources=%s", cfg.FrontendURI, resources)

	// Append environment if explicitly provided or resolvable from app context
	envID := environmentFlag

	if envID == "" {
		appInfo, err := utils.GetApplicationInfo("")
		if err == nil {
			apiClient := singletons.GetAPIClient()

			envResp, err := apiClient.GetApplicationEnvironment(appInfo.ApplicationID)
			if err == nil && envResp.EnvironmentID != nil {
				envID = *envResp.EnvironmentID
			}
		}
	}

	if envID != "" {
		connectURL += fmt.Sprintf("&environmentId=%s", envID)
	}

	if err := utils.OpenBrowser(connectURL); err != nil {
		cmd.Printf("Failed to open browser automatically. Please visit:\n%s\n", connectURL)
		return nil
	}

	cmd.Printf("Opening connect page in your browser:\n%s\n", connectURL)
	return nil
}
