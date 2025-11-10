package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

	// Determine the desired directory name (based on app name)
	desiredDir := sanitizeDirName(selectedApp.Name)

	// Determine the repository directory (use the repository name for git operations)
	repoDir := filepath.Join(".", selectedApp.GithubRepositoryName)

	// Check if either directory exists
	var gitErr error
	var workingDir string

	// Check if desired directory exists
	if _, err := os.Stat(desiredDir); err == nil {
		// Desired directory exists, use it for pulling
		workingDir = desiredDir
		cmd.Printf("Directory '%s' already exists. Pulling latest changes...\n", workingDir)
		gitErr = git.Pull(workingDir)
	} else if _, err := os.Stat(repoDir); err == nil {
		// Repository directory exists (old naming), use it for pulling then rename
		workingDir = repoDir
		cmd.Printf("Directory '%s' already exists. Pulling latest changes...\n", workingDir)
		gitErr = git.Pull(workingDir)
	} else {
		// Neither directory exists, clone directly to desired directory
		workingDir = desiredDir
		cmd.Printf("Directory '%s' does not exist. Cloning repository...\n", workingDir)
		_, gitErr = cloneRepository(selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS, workingDir)
	}

	// Handle git authentication errors
	if gitErr != nil {
		if isGitAuthError(gitErr) {
			// Ensure repository access
			if err := ensureRepositoryAccess(cmd, selectedApp.ID, selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS); err != nil {
				return fmt.Errorf("failed to ensure repository access: %w", err)
			}

			// Retry the git operation
			if _, err := os.Stat(workingDir); err == nil {
				// Directory exists, pull
				cmd.Printf("Pulling latest changes...\n")
				gitErr = git.Pull(workingDir)
			} else {
				// Directory doesn't exist, clone
				cmd.Printf("Cloning repository...\n")
				_, gitErr = cloneRepository(selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS, workingDir)
			}

			// Check if retry succeeded
			if gitErr != nil {
				return fmt.Errorf("failed to access repository after accepting invitation: %w", gitErr)
			}
		} else {
			return fmt.Errorf("failed to pull repository: %w", gitErr)
		}
	}

	// Rename directory if needed
	finalDir := workingDir
	if workingDir != desiredDir {
		// Check if desired directory name is available
		if _, err := os.Stat(desiredDir); err == nil {
			// Desired directory already exists, keep using working directory
			cmd.Printf("\nNote: Directory '%s' already exists, keeping repository in '%s'\n", desiredDir, workingDir)
		} else if os.IsNotExist(err) {
			// Rename to desired directory
			if err := os.Rename(workingDir, desiredDir); err != nil {
				cmd.Printf("\nWarning: Failed to rename directory to '%s': %v\n", desiredDir, err)
				cmd.Printf("Continuing with directory '%s'\n", workingDir)
			} else {
				cmd.Printf("\nRenamed directory from '%s' to '%s'\n", workingDir, desiredDir)
				finalDir = desiredDir
			}
		} else {
			// Some other error checking the desired directory
			cmd.Printf("\nWarning: Could not check directory '%s': %v\n", desiredDir, err)
			cmd.Printf("Continuing with directory '%s'\n", workingDir)
		}
	}

	// Generate env file
	cmd.Println("\nGenerating .env file...")
	envFilePath, _, err := generateEnvFile(finalDir)
	if err != nil {
		return fmt.Errorf("failed to generate .env file: %w", err)
	}
	cmd.Printf("Successfully generated .env file at: %s\n", envFilePath)

	// Generate resources file
	cmd.Println("\nGenerating RESOURCES.md file...")
	resourcesFilePath, _, err := generateResourcesFile(finalDir)
	if err != nil {
		return fmt.Errorf("failed to generate RESOURCES.md file: %w", err)
	}
	cmd.Printf("Successfully generated RESOURCES.md file at: %s\n", resourcesFilePath)

	cmd.Println("\n✓ Application pull complete!")

	printSuccessMessage(cmd, selectedApp.Name)
	return nil
}

// sanitizeDirName converts an application name to a valid directory name
func sanitizeDirName(name string) string {
	// Convert to lowercase
	dirName := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	dirName = reg.ReplaceAllString(dirName, "-")

	// Remove leading/trailing hyphens
	dirName = strings.Trim(dirName, "-")

	// If the result is empty, use a default
	if dirName == "" {
		dirName = "app"
	}

	return dirName
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
