package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// cloneCmd represents the app clone command
var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone an application repository",
	Long:  `Select and clone an application repository from your organization, then generate env and resources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClone(cmd)
	},
}

func runClone(cmd *cobra.Command) error {
	// Get the default organization ID from keyring
	orgID, orgName, err := token.GetDefaultOrg()
	if err != nil {
		return errors.WrapError("failed to get default organization", errors.ErrorNoOrganizationSelected)
	}

	cmd.Printf("Fetching applications for organization: %s\n", orgName)

	// Get API client
	apiClient := singletons.GetAPIClient()

	// Get applications for the organization
	appsResp, err := apiClient.GetOrganizationApplications(orgID)
	if err != nil {
		return errors.WrapError("failed to get applications", err)
	}

	if len(appsResp.Applications) == 0 {
		return errors.ErrorNoApplicationsAvailable
	}

	// Let user select application
	selectedApp, err := selectApplication(cmd, appsResp.Applications)
	if err != nil {
		return errors.WrapError("failed to select application", err)
	}

	cmd.Printf("Selected application: %s\n", selectedApp.Name)

	// Determine the desired directory name (based on app name)
	desiredDir := sanitizeDirName(selectedApp.Name)

	// Determine the repository directory (use the repository name for git operations)
	repoDir := filepath.Join(".", selectedApp.GithubRepositoryName)

	// Determine which directory to use (prefer desiredDir, fall back to repoDir if it exists)
	var workingDir string
	if _, err := os.Stat(desiredDir); err == nil {
		workingDir = desiredDir
	} else if _, err := os.Stat(repoDir); err == nil {
		workingDir = repoDir
	} else {
		workingDir = desiredDir
	}

	// Ensure the directory is a properly configured git repository
	gitErr := ensureGitRepository(cmd, workingDir, selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS)

	// Handle git authentication errors
	if gitErr != nil {
		if isGitAuthError(gitErr) {
			// Ensure repository access
			if err := utils.EnsureRepositoryAccess(cmd, selectedApp.ID, selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS); err != nil {
				return errors.WrapError("failed to ensure repository access", err)
			}

			// For some reason, there's a race where the repo is still not available for clones
			gitErr = ensureGitRepositoryWithRetries(cmd, workingDir, selectedApp.CloneURLSSH, selectedApp.CloneURLHTTPS)
			// Check if retry succeeded
			if gitErr != nil {
				return errors.ErrorGitRepositoryAccessFailed
			}
		} else {
			return errors.ErrorGitCloneFailed
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
		return errors.WrapError("failed to generate .env file", err)
	}
	cmd.Printf("Successfully generated .env file at: %s\n", envFilePath)

	cmd.Println("\n✓ Application clone complete!")

	printSuccessMessage(cmd, finalDir)
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
		return nil, errors.ErrorNoApplicationsAvailable
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
				Title("Select an application to clone").
				Options(options...).
				Value(&selectedID),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, errors.WrapError("failed to get selection", err)
	}

	// Find the selected application
	for i, app := range apps {
		if app.ID == selectedID {
			return &apps[i], nil
		}
	}

	return nil, errors.ErrorApplicationNotFound
}
