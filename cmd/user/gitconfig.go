package user

import (
	"errors"

	"github.com/charmbracelet/huh"
	mjrToken "github.com/major-technology/cli/clients/token"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/spf13/cobra"
)

// gitconfigCmd represents the gitconfig command
var gitconfigCmd = &cobra.Command{
	Use:   "gitconfig",
	Short: "Configure git settings",
	Long:  `Configure git-related settings such as your GitHub username.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runGitConfig(cobraCmd)
	},
}

func runGitConfig(cobraCmd *cobra.Command) error {
	// Get current GitHub username if it exists
	currentUsername, err := mjrToken.GetGithubUsername()
	if err != nil {
		return clierrors.WrapError("failed to get current GitHub username", err)
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
						return errors.New("GitHub username is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return clierrors.WrapError("failed to collect GitHub username", err)
	}

	// Store the GitHub username
	if err := mjrToken.StoreGithubUsername(githubUsername); err != nil {
		return clierrors.WrapError("failed to store GitHub username", err)
	}

	cobraCmd.Printf("✓ GitHub username saved: %s\n", githubUsername)
	return nil
}
