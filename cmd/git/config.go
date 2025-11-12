package git

import (
	"fmt"

	"github.com/charmbracelet/huh"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

// configCmd represents the git config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure git settings",
	Long:  `Configure git-related settings such as your GitHub username.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runConfig(cobraCmd))
	},
}

func runConfig(cobraCmd *cobra.Command) error {
	// Get current GitHub username if it exists
	currentUsername, err := mjrToken.GetGithubUsername()
	if err != nil {
		return fmt.Errorf("failed to get current GitHub username: %w", err)
	}

	// Show current username if it exists
	if currentUsername != "" {
		cobraCmd.Printf("Current GitHub username: %s\n\n", currentUsername)
	}

	// Prompt for new GitHub username
	var githubUsername string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("GitHub Username").
				Description("Enter your GitHub username").
				Value(&githubUsername).
				Placeholder(currentUsername).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("GitHub username is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to collect GitHub username: %w", err)
	}

	// Store the GitHub username
	if err := mjrToken.StoreGithubUsername(githubUsername); err != nil {
		return fmt.Errorf("failed to store GitHub username: %w", err)
	}

	cobraCmd.Printf("✓ GitHub username saved: %s\n", githubUsername)
	return nil
}

