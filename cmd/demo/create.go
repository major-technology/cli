package demo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var DemoCloneURLSSH = "git@github.com:major-technology/vite-api-usage-demo.git"
var DemoCloneURLHTTPS = "https://github.com/major-technology/vite-api-usage-demo.git"

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new demo application",
	Long:  `Create a new demo application with a GitHub repository and the demo template.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runCreate(cobraCmd)
	},
}

func runCreate(cobraCmd *cobra.Command) error {
	// Get default org from keychain
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return errors.ErrorNoOrganizationSelected
	}

	cobraCmd.Printf("Creating demo application in organization: %s\n\n", orgName)

	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Call POST /demo_application
	createResp, err := apiClient.CreateDemoApplication(orgID)
	if err != nil {
		return err
	}

	cobraCmd.Printf("✓ Demo application created with ID: %s\n", createResp.ApplicationID)
	cobraCmd.Printf("✓ Repository: %s\n", createResp.RepositoryName)

	// Check if we have permissions to use SSH or HTTPS
	useSSH := false
	if utils.CanUseSSH() {
		cobraCmd.Println("✓ SSH access detected")
		useSSH = true
	} else if createResp.CloneURLHTTPS != "" {
		cobraCmd.Println("✓ Using HTTPS for git operations")
		useSSH = false
	} else {
		return errors.ErrorNoValidCloneMethodAvailable
	}

	// Determine which clone URL to use
	templateURL := DemoCloneURLHTTPS
	if useSSH {
		templateURL = DemoCloneURLSSH
	}

	cloneURL := createResp.CloneURLHTTPS
	if useSSH {
		cloneURL = createResp.CloneURLSSH
	}

	// Create a temporary directory for the template
	tempDir, err := os.MkdirTemp("", "major-demo-template-*")
	if err != nil {
		return errors.WrapError("failed to create temp directory", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone the hardcoded demo template repository
	if err := git.Clone(templateURL, tempDir); err != nil {
		return errors.WrapError("failed to clone demo template repository", err)
	}

	cobraCmd.Println("✓ Demo template cloned")

	// Remove the existing remote origin
	if err := git.RemoveRemote(tempDir, "origin"); err != nil {
		return errors.WrapError("failed to remove remote origin", err)
	}

	cobraCmd.Println("✓ Removed template remote")

	// Add the new remote
	if err := git.AddRemote(tempDir, "origin", cloneURL); err != nil {
		return errors.WrapError("failed to add new remote", err)
	}

	cobraCmd.Printf("✓ Added new remote: %s\n", cloneURL)

	// Ensure repository access before pushing
	if err := utils.EnsureRepositoryAccess(cobraCmd, createResp.ApplicationID, createResp.CloneURLSSH, createResp.CloneURLHTTPS); err != nil {
		return errors.WrapError("failed to ensure repository access", err)
	}

	// Get the demo resource
	cobraCmd.Println("\nFetching demo resource...")
	demoResourceResp, err := apiClient.GetDemoResource(orgID)
	if err != nil {
		return errors.WrapError("failed to get demo resource", err)
	}

	var selectedResources []api.ResourceItem
	if demoResourceResp.Resource != nil {
		selectedResources = []api.ResourceItem{*demoResourceResp.Resource}
		cobraCmd.Printf("✓ Demo resource found: %s (%s)\n", demoResourceResp.Resource.Name, demoResourceResp.Resource.Type)

		// Save the demo resource to the application
		_, err = apiClient.SaveApplicationResources(orgID, createResp.ApplicationID, []string{demoResourceResp.Resource.ID})
		if err != nil {
			return errors.WrapError("failed to save demo resource", err)
		}
		cobraCmd.Println("✓ Demo resource linked to application")
	}

	// Push to the new remote
	if err := git.Push(tempDir); err != nil {
		return errors.WrapError("failed to push to new repository", err)
	}

	// Move the repository to the current directory
	targetDir := filepath.Join(".", createResp.RepositoryName)
	if err := os.Rename(tempDir, targetDir); err != nil {
		return errors.WrapError("failed to move repository", err)
	}

	cobraCmd.Printf("\n✓ Demo application '%s' successfully created in ./%s\n", createResp.RepositoryName, createResp.RepositoryName)
	cobraCmd.Printf("  Clone URL: %s\n", cloneURL)

	// Add the demo resource to the project
	if len(selectedResources) > 0 {
		if err := utils.AddResourcesToProject(cobraCmd, targetDir, selectedResources, createResp.ApplicationID); err != nil {
			return errors.ErrorFailedToSelectResources
		}
	}

	// Generate .env file
	cobraCmd.Println("\nGenerating .env file...")
	envFilePath, envVars, err := generateEnvFile(targetDir, orgID, createResp.ApplicationID)
	if err != nil {
		cobraCmd.Printf("Warning: Failed to generate .env file: %v\n", err)
	} else {
		cobraCmd.Printf("✓ Generated .env file at: %s\n", envFilePath)

		// Generate .mcp.json for Claude Code
		if _, err := utils.GenerateMcpConfig(targetDir, envVars); err != nil {
			cobraCmd.Printf("Warning: Failed to generate .mcp.json: %v\n", err)
		} else {
			cobraCmd.Println("✓ Generated .mcp.json for Claude Code")
		}
	}

	printSuccessMessage(cobraCmd, createResp.RepositoryName)

	return nil
}

// printSuccessMessage displays a nicely formatted success message with next steps
func printSuccessMessage(cobraCmd *cobra.Command, appName string) {
	// Define styles
	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")). // Green
		MarginTop(1).
		MarginBottom(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")). // Cyan
		Bold(true)

	descriptionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")) // Gray

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")). // Blue
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	cdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Green
		Bold(true)

	// Build the message
	successMsg := successStyle.Render("🎉 Congrats on setting up your demo app!")

	// CD instruction
	cdInstruction := cdStyle.Render(fmt.Sprintf("First, navigate to your app directory:\n  cd %s", appName))

	nextStepsTitle := titleStyle.Render("What's next?")

	// Commands with improved descriptions
	startCommand := commandStyle.Render("major app start")
	startDesc := descriptionStyle.Render("  Start your app locally for development")

	deployCommand := commandStyle.Render("major app deploy")
	deployDesc := descriptionStyle.Render("  Deploy your app to production when ready")

	content := fmt.Sprintf("%s\n\n%s\n\n%s\n%s\n\n%s\n%s",
		nextStepsTitle,
		cdInstruction,
		startCommand,
		startDesc,
		deployCommand,
		deployDesc,
	)

	box := boxStyle.Render(content)

	// Print everything
	cobraCmd.Println(successMsg)
	cobraCmd.Println(box)
}

// generateEnvFile generates a .env file for the application in the specified directory.
func generateEnvFile(targetDir, orgID, applicationID string) (string, map[string]string, error) {
	apiClient := singletons.GetAPIClient()

	envVars, err := apiClient.GetApplicationEnv(orgID, applicationID)
	if err != nil {
		return "", nil, errors.WrapError("failed to get environment variables", err)
	}

	// Create .env file path
	envFilePath := filepath.Join(targetDir, ".env")

	// Build the .env file content
	var envContent strings.Builder
	for key, value := range envVars {
		envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Write to .env file
	err = os.WriteFile(envFilePath, []byte(envContent.String()), 0644)
	if err != nil {
		return "", nil, errors.WrapError("failed to write .env file", err)
	}

	return envFilePath, envVars, nil
}
