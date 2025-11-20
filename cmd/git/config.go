package git

import (
	"github.com/charmbracelet/huh"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	pkgErrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// configCmd represents the git config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure git settings",
	Long:  `Configure git-related settings such as your GitHub username.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runConfig(cobraCmd)
	},
}

func runConfig(cobraCmd *cobra.Command) error {
	// Get current GitHub username if it exists
	currentUsername, err := mjrToken.GetGithubUsername()
	if err != nil {
		return errors.WrapError("failed to get current GitHub username", err)
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
						return pkgErrors.New("GitHub username is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return errors.WrapError("failed to collect GitHub username", err)
	}

	// Store the GitHub username
	if err := mjrToken.StoreGithubUsername(githubUsername); err != nil {
		return errors.WrapError("failed to store GitHub username", err)
	}

	cobraCmd.Printf("✓ GitHub username saved: %s\n", githubUsername)
	return nil
}
