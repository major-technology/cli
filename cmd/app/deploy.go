package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new version of the application",
	Long:  `Creates a new version by committing and pushing changes, then deploying to the platform.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runDeploy(cobraCmd))
	},
}

func runDeploy(cobraCmd *cobra.Command) error {
	// Check if we're in a git repository
	if !git.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Check for uncommitted changes
	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if hasChanges {
		cobraCmd.Println("📝 Uncommitted changes detected")

		// Prompt for commit message
		var commitMessage string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("Commit Message").
					Description("Enter a commit message for your changes").
					Value(&commitMessage).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("commit message is required")
						}
						return nil
					}),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("failed to collect commit message: %w", err)
		}

		// Stage all changes
		cobraCmd.Println("\nStaging changes...")
		if err := git.Add(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
		cobraCmd.Println("✓ Changes staged")

		// Commit changes
		cobraCmd.Println("Committing changes...")
		if err := git.Commit(commitMessage); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}
		cobraCmd.Println("✓ Changes committed")

		// Push to remote
		cobraCmd.Println("Pushing to remote...")
		if err := git.PushToMain(); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
		cobraCmd.Println("✓ Changes pushed to remote")
	} else {
		cobraCmd.Println("✓ No uncommitted changes")
	}

	// Get application ID
	cobraCmd.Println("\nDeploying new version...")
	applicationID, err := getApplicationID()
	if err != nil {
		return fmt.Errorf("failed to get application ID: %w", err)
	}

	// Call API to create new version
	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.CreateApplicationVersion(applicationID)
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return err
	}

	cobraCmd.Printf("\n✓ Version deployed successfully!\n")
	cobraCmd.Printf("  Version ID: %s\n", resp.VersionID)

	return nil
}
