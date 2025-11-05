package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
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
	if _, err := os.Stat(targetDir); err == nil {
		// Directory exists, just pull
		cmd.Printf("Directory '%s' already exists. Pulling latest changes...\n", targetDir)

		if err := git.Pull(targetDir); err != nil {
			return fmt.Errorf("failed to pull repository: %w", err)
		}

		cmd.Println("Successfully pulled latest changes")
	} else if os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' does not exist. Please clone the repository first", targetDir)
	} else {
		return fmt.Errorf("failed to check directory: %w", err)
	}

	// Generate env file
	cmd.Println("\nGenerating .env file...")
	envFilePath, numVars, err := generateEnvFile(targetDir)
	if err != nil {
		return fmt.Errorf("failed to generate .env file: %w", err)
	}
	cmd.Printf("Successfully generated .env file at: %s\n", envFilePath)
	cmd.Printf("Environment variables written: %d\n", numVars)

	// Generate resources file
	cmd.Println("\nGenerating RESOURCES.md file...")
	resourcesFilePath, numResources, err := generateResourcesFile(targetDir)
	if err != nil {
		return fmt.Errorf("failed to generate RESOURCES.md file: %w", err)
	}
	cmd.Printf("Successfully generated RESOURCES.md file at: %s\n", resourcesFilePath)
	cmd.Printf("Resources written: %d\n", numResources)

	cmd.Println("\n✓ Application pull complete!")

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
