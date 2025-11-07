package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// pullCmd represents the app pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull an application repository",
	Long:  `Select and pull an application repository from your organization, then generate env and resources.`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runPull(cmd))
	},
}

func runPull(cmd *cobra.Command) error {
	// Get the default organization ID from keyring
	orgID, orgName, err := token.GetDefaultOrg()
	if err != nil {
		return fmt.Errorf("failed to get default organization: %w\nPlease run 'major org select' to set a default organization", err)
	}

	cmd.Printf("Fetching applications for organization: %s\n", orgName)

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	// Get applications for the organization
	appsResp, err := apiClient.GetOrganizationApplications(orgID)
	if err != nil {
		return fmt.Errorf("failed to get applications: %w", err)
	}

	if len(appsResp.Applications) == 0 {
		cmd.Println("No applications available for this organization")
		return nil
	}

	// Let user select application
	selectedApp, err := selectApplication(cmd, appsResp.Applications)
	if err != nil {
		return fmt.Errorf("failed to select application: %w", err)
	}

	cmd.Printf("Selected application: %s\n", selectedApp.Name)

	// Determine the target directory (use the repository name)
	targetDir := filepath.Join(".", selectedApp.GithubRepositoryName)

	// Check if the directory already exists
	var gitErr error
	if _, err := os.Stat(targetDir); err == nil {
		// Directory exists, just pull
		cmd.Printf("Directory '%s' already exists. Pulling latest changes...\n", targetDir)

		gitErr = git.Pull(targetDir)
	} else if os.IsNotExist(err) {
		// Directory doesn't exist, clone it
		cmd.Printf("Directory '%s' does not exist. Cloning repository...\n", targetDir)

		_, gitErr = cloneRepository(selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS, targetDir)
	} else {
		return fmt.Errorf("failed to check directory: %w", err)
	}

	// Handle git authentication errors
	if gitErr != nil {
		if isGitAuthError(gitErr) {
			cmd.Printf("Repository access required.\n\n")

			// Prompt for GitHub username
			var githubUsername string
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("What is your GitHub username?").
						Value(&githubUsername).
						Validate(func(s string) error {
							if s == "" {
								return fmt.Errorf("GitHub username is required")
							}
							return nil
						}),
				),
			)

			if err := form.Run(); err != nil {
				return fmt.Errorf("failed to get GitHub username: %w", err)
			}

			cmd.Printf("\nAdding @%s as a collaborator to the repository...\n", githubUsername)

			// Add user as GitHub collaborator
			_, err := apiClient.AddGithubCollaborators(selectedApp.ID, githubUsername)
			if err != nil {
				return fmt.Errorf("failed to add GitHub collaborator: %w", err)
			}

			cmd.Println("✓ Invitation sent!")

			// Try to extract and open the GitHub repository URL
			cloneURL := selectedApp.CloneURLHTTPS
			if cloneURL == "" {
				cloneURL = selectedApp.CloneURLSSH
			}

			githubURL, urlErr := extractGitHubURL(cloneURL)
			if urlErr == nil {
				cmd.Printf("\nPlease accept the invitation at: %s\n", githubURL)
				_ = utils.OpenBrowser(githubURL)
				cmd.Printf("You may need to refresh the page to see the invitation.\n")
			}

			// Poll for repository access
			if !pollForRepositoryAccess(cmd, selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS) {
				return fmt.Errorf("timeout waiting for repository access - please try again after accepting the invitation")
			}

			cmd.Println("\n✓ Repository access granted!")

			// Retry the git operation
			if _, err := os.Stat(targetDir); err == nil {
				// Directory exists, pull
				cmd.Printf("Pulling latest changes...\n")
				gitErr = git.Pull(targetDir)
			} else if os.IsNotExist(err) {
				// Directory doesn't exist, clone
				cmd.Printf("Cloning repository...\n")
				_, gitErr = cloneRepository(selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS, targetDir)
			}

			// Check if retry succeeded
			if gitErr != nil {
				return fmt.Errorf("failed to access repository after accepting invitation: %w", gitErr)
			}
		} else {
			return fmt.Errorf("failed to pull repository: %w", gitErr)
		}
	}

	// Generate env file
	cmd.Println("\nGenerating .env file...")
	envFilePath, _, err := generateEnvFile(targetDir)
	if err != nil {
		return fmt.Errorf("failed to generate .env file: %w", err)
	}
	cmd.Printf("Successfully generated .env file at: %s\n", envFilePath)

	// Generate resources file
	cmd.Println("\nGenerating RESOURCES.md file...")
	resourcesFilePath, _, err := generateResourcesFile(targetDir)
	if err != nil {
		return fmt.Errorf("failed to generate RESOURCES.md file: %w", err)
	}
	cmd.Printf("Successfully generated RESOURCES.md file at: %s\n", resourcesFilePath)

	cmd.Println("\n✓ Application pull complete!")

	printSuccessMessage(cmd, selectedApp.Name)
	return nil
}

// selectApplication prompts the user to select an application from the list
func selectApplication(cmd *cobra.Command, apps []api.ApplicationItem) (*api.ApplicationItem, error) {
	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications available")
	}

	// If only one application, automatically select it
	if len(apps) == 1 {
		cmd.Printf("Only one application available. Automatically selecting it.\n")
		return &apps[0], nil
	}

	// Create options for huh select
	options := make([]huh.Option[string], len(apps))
	for i, app := range apps {
		options[i] = huh.NewOption(app.Name, app.ID)
	}

	var selectedID string

	// Create and run the select form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select an application to pull").
				Options(options...).
				Value(&selectedID),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get selection: %w", err)
	}

	// Find the selected application
	for i, app := range apps {
		if app.ID == selectedID {
			return &apps[i], nil
		}
	}

	return nil, fmt.Errorf("selected application not found")
}

// pollForRepositoryAccess polls the repository to check if access has been granted
// Polls every 2 seconds with a 5 minute timeout
// Returns true if access is granted, false if timeout
func pollForRepositoryAccess(cmd *cobra.Command, sshURL, httpsURL string) bool {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			if checkRepositoryAccess(sshURL, httpsURL) {
				return true
			}
			cmd.Print(".")
		}
	}
}
