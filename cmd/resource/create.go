package resource

import (
	"fmt"

	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// createCmd represents the resource create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Open the resource creation page in your browser",
	Long:  `Open the resource creation page in your default browser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCreate(cmd)
	},
}

func runCreate(cmd *cobra.Command) error {
	// Get config to access frontend URI
	cfg := singletons.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not initialized")
	}

	// Construct the resource creation URL
	resourceURL := fmt.Sprintf("%s/resources?action=add", cfg.FrontendURI)

	// Open the URL in the browser
	if err := utils.OpenBrowser(resourceURL); err != nil {
		// If browser fails to open, still show the URL
		cmd.Printf("Failed to open browser automatically. Please visit:\n%s\n", resourceURL)
		return nil
	}

	cmd.Printf("Opening resource creation page in your browser:\n%s\n", resourceURL)
	return nil
}
