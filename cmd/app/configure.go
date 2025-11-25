package app

import (
	"fmt"

	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Open the app configurations in your browser",
	Long:  `Open the app configurations in your default browser for the current application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigure(cmd)
	},
}

func runConfigure(cmd *cobra.Command) error {
	// Get application ID
	applicationID, err := getApplicationID()
	if err != nil {
		return err
	}

	// Get config to access frontend URI
	cfg := singletons.GetConfig()

	// Construct the app settings URL
	configureURL := fmt.Sprintf("%s/home?dialog=app-settings&appId=%s", cfg.FrontendURI, applicationID)

	// Open the URL in the browser
	if err := utils.OpenBrowser(configureURL); err != nil {
		// If browser fails to open, still show the URL
		cmd.Printf("Failed to open browser automatically. Please visit:\n%s\n", configureURL)
		return nil
	}

	cmd.Printf("Opening app configurations in your browser:\n%s\n", configureURL)
	return nil
}

