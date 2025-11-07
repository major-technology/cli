package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

const templateRepoURL = "https://github.com/major-technology/basic-template.git"

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	Long:  `Create a new application with a GitHub repository and sets up the basic template.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runCreate(cobraCmd))
	},
}

func runCreate(cobraCmd *cobra.Command) error {
	// Get default org from keychain
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return fmt.Errorf("no default organization set. Please run 'major user login' first: %w", err)
	}

	cobraCmd.Printf("Creating application in organization: %s\n\n", orgName)

	// Ask user for application name and description
	var appName, appDescription string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Application Name").
				Description("Enter a name for your application").
				Value(&appName).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("application name is required")
					}
					return nil
				}),
			huh.NewText().
				Title("Application Description").
				Description("Enter a description for your application").
				Value(&appDescription).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("application description is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to collect application details: %w", err)
	}

	cobraCmd.Printf("\nCreating application '%s'...\n", appName)

	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Call POST /applications (token will be fetched automatically)
	createResp, err := apiClient.CreateApplication(appName, appDescription, orgID)
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return err
	}

	cobraCmd.Printf("✓ Application created with ID: %s\n", createResp.ApplicationID)
	cobraCmd.Printf("✓ Repository: %s\n", createResp.RepositoryName)

	// Check if we have permissions to use SSH or HTTPS
	useSSH := false
	if canUseSSH() {
		cobraCmd.Println("✓ SSH access detected")
		useSSH = true
	} else if createResp.CloneURLHTTPS != "" {
		cobraCmd.Println("✓ Using HTTPS for git operations")
		useSSH = false
	} else {
		return fmt.Errorf("no valid clone method available")
	}

	// Determine which clone URL to use
	cloneURL := createResp.CloneURLHTTPS
	if useSSH {
		cloneURL = createResp.CloneURLSSH
	}

	// Create a temporary directory for the template
	tempDir, err := os.MkdirTemp("", "major-template-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cobraCmd.Printf("\nCloning template repository...\n")

	// Clone the template repository
	if err := git.Clone(templateRepoURL, tempDir); err != nil {
		return fmt.Errorf("failed to clone template repository: %w", err)
	}

	cobraCmd.Println("✓ Template cloned")

	// Remove the existing remote origin
	if err := git.RemoveRemote(tempDir, "origin"); err != nil {
		return fmt.Errorf("failed to remove remote origin: %w", err)
	}

	cobraCmd.Println("✓ Removed template remote")

	// Add the new remote
	if err := git.AddRemote(tempDir, "origin", cloneURL); err != nil {
		return fmt.Errorf("failed to add new remote: %w", err)
	}

	cobraCmd.Printf("✓ Added new remote: %s\n", cloneURL)

	// Ensure repository access before pushing
	if err := ensureRepositoryAccess(cobraCmd, createResp.ApplicationID, createResp.CloneURLSSH, createResp.CloneURLHTTPS); err != nil {
		return fmt.Errorf("failed to ensure repository access: %w", err)
	}

	// Push to the new remote
	cobraCmd.Println("\nPushing to new repository...")
	if err := git.Push(tempDir); err != nil {
		return fmt.Errorf("failed to push to new repository: %w", err)
	}

	cobraCmd.Println("✓ Pushed to repository")

	// Move the repository to the current directory
	targetDir := filepath.Join(".", appName)
	if err := os.Rename(tempDir, targetDir); err != nil {
		return fmt.Errorf("failed to move repository: %w", err)
	}

	cobraCmd.Printf("\n✓ Application '%s' successfully created in ./%s\n", appName, appName)
	cobraCmd.Printf("  Clone URL: %s\n", cloneURL)

	// Generate .env file
	cobraCmd.Println("\nGenerating .env file...")
	envFilePath, _, err := generateEnvFile(targetDir)
	if err != nil {
		cobraCmd.Printf("Warning: Failed to generate .env file: %v\n", err)
	} else {
		cobraCmd.Printf("✓ Generated .env file at: %s\n", envFilePath)
	}

	// Generate RESOURCES.md file
	cobraCmd.Println("\nGenerating RESOURCES.md file...")
	resourcesFilePath, _, err := generateResourcesFile(targetDir)
	if err != nil {
		cobraCmd.Printf("Warning: Failed to generate RESOURCES.md file: %v\n", err)
	} else {
		cobraCmd.Printf("✓ Generated RESOURCES.md file at: %s\n", resourcesFilePath)
	}

	printSuccessMessage(cobraCmd, appName)

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
		Foreground(lipgloss.Color("11")). // Yellow
		Bold(true).
		MarginTop(1).
		MarginBottom(1)

	// Build the message
	successMsg := successStyle.Render("🎉 Congrats on setting up your app!")

	// CD instruction
	cdInstruction := cdStyle.Render(fmt.Sprintf("First, navigate to your app directory:\n  cd %s", appName))

	nextStepsTitle := titleStyle.Render("What's next?")

	// Commands with improved descriptions
	startCommand := commandStyle.Render("major app start")
	startDesc := descriptionStyle.Render("  Start your app locally for development")

	deployCommand := commandStyle.Render("major app deploy")
	deployDesc := descriptionStyle.Render("  Deploy your app to production when ready")

	editorCommand := commandStyle.Render("major app editor")
	editorDesc := descriptionStyle.Render("  Open your app in the UI editor")

	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s",
		nextStepsTitle,
		startCommand,
		startDesc,
		deployCommand,
		deployDesc,
		editorCommand,
		editorDesc,
	)

	box := boxStyle.Render(content)

	// Print everything
	cobraCmd.Println(successMsg)
	cobraCmd.Println(cdInstruction)
	cobraCmd.Println(box)
}
