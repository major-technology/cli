package app

import (
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display information about the current application",
	Long:  `Display information about the application in the current directory, including the application ID.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInfo(cmd)
	},
}

func runInfo(cmd *cobra.Command) error {
	// Get application ID
	applicationID, err := getApplicationID()
	if err != nil {
		return err
	}

	// Print only the application ID
	cmd.Printf("Application ID: %s\n", applicationID)

	return nil
}
