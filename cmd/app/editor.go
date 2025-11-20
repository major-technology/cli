package app

import (
	"fmt"

	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// editCmd represents the editor command
var editCmd = &cobra.Command{
	Use:   "editor",
	Short: "Open the application editor in your browser",
	Long:  `Open the application editor in your default browser for the current application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(cmd)
	},
}

func runEdit(cmd *cobra.Command) error {
	// Get application ID
	applicationID, err := getApplicationID()
	if err != nil {
		return err
	}

	// Get config to access frontend URI
	cfg := singletons.GetConfig()

	// Construct the editor URL
	editorURL := fmt.Sprintf("%s/apps/%s/edit", cfg.FrontendURI, applicationID)

	// Open the URL in the browser
	if err := utils.OpenBrowser(editorURL); err != nil {
		// If browser fails to open, still show the URL
		cmd.Printf("Failed to open browser automatically. Please visit:\n%s\n", editorURL)
		return nil
	}

	cmd.Printf("Opening application editor in your browser:\n%s\n", editorURL)
	return nil
}
