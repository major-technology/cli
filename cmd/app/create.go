package app

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Flag variables for non-interactive mode
var (
	flagAppName        string
	flagAppDescription string
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	Long: `Create a new NextJS application with a GitHub repository.

By default, this command runs interactively, prompting for application name and description.
You can also provide these values via flags for non-interactive usage:

  major app create --name "my-app" --description "My application"

GitHub username is auto-detected from your SSH configuration.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runCreate(cobraCmd)
	},
}

func init() {
	createCmd.Flags().StringVar(&flagAppName, "name", "", "Application name (skips interactive prompt)")
	createCmd.Flags().StringVar(&flagAppDescription, "description", "", "Application description (skips interactive prompt)")
	createCmd.Flags().StringVar(&flagGithubUser, "github-user", "", "GitHub username for repository access (for non-interactive mode)")
}

func runCreate(cobraCmd *cobra.Command) error {
	// Get default org from keychain
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return errors.ErrorNoOrganizationSelected
	}

	cobraCmd.Printf("Creating application in organization: %s\n\n", orgName)

	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Use flag values if provided, otherwise prompt interactively
	appName := flagAppName
	appDescription := flagAppDescription
	var selectedThemeID string

	// Check if we need to prompt for any values
	needsPrompt := appName == "" || appDescription == ""

	if needsPrompt {
		// Build form fields only for missing values
		var formFields []huh.Field

		if appName == "" {
			formFields = append(formFields,
				huh.NewInput().
					Title("Application Name").
					Description("Enter a name for your application").
					Value(&appName).
					Validate(func(s string) error {
						if s == "" {
							return errors.ErrorApplicationNameRequired
						}
						return nil
					}),
			)
		}

		if appDescription == "" {
			formFields = append(formFields,
				huh.NewInput().
					Title("Application Description").
					Description("Enter a description for your application").
					Value(&appDescription).
					Validate(func(s string) error {
						if s == "" {
							return errors.ErrorApplicationDescriptionRequired
						}
						return nil
					}),
			)
		}

		// Add theme selection field
		themeField, themeErr := buildThemeSelectField(apiClient, orgID, &selectedThemeID)

		if themeErr != nil {
			cobraCmd.Printf("Warning: Failed to load themes: %v\n", themeErr)
		}

		if themeField != nil {
			formFields = append(formFields, themeField)
		}

		form := huh.NewForm(huh.NewGroup(formFields...))

		if err := form.Run(); err != nil {
			return errors.WrapError("failed to collect application details", err)
		}
	}

	// Validate that we have both values (in case user provided only one flag)
	if appName == "" {
		return errors.ErrorApplicationNameRequired
	}

	if appDescription == "" {
		return errors.ErrorApplicationDescriptionRequired
	}

	// Convert theme ID to pointer for API call
	var themeIDPtr *string

	if selectedThemeID != "" {
		themeIDPtr = &selectedThemeID
	}

	cobraCmd.Printf("\nCreating application '%s'...\n", appName)

	createResp, err := apiClient.CreateApplication(appName, appDescription, orgID, themeIDPtr)
	if err != nil {
		return err
	}

	cobraCmd.Printf("✓ Application created with ID: %s\n", createResp.ApplicationID)
	cobraCmd.Printf("✓ Repository: %s\n", createResp.RepositoryName)

	// Ensure repository access before cloning
	// Use non-interactive mode if all required flags were provided
	isNonInteractive := flagAppName != "" && flagAppDescription != ""
	opts := utils.EnsureRepositoryAccessOptions{
		NonInteractive: isNonInteractive,
		GithubUsername: flagGithubUser,
	}
	err = utils.EnsureRepositoryAccessWithOptions(cobraCmd, createResp.ApplicationID, createResp.CloneURLSSH, createResp.CloneURLHTTPS, opts)

	// Check if invitation is pending (user needs to accept)
	if invErr, ok := err.(*utils.InvitationPendingError); ok {
		cobraCmd.Println("")
		cobraCmd.Println("╭─────────────────────────────────────────────────────────────╮")
		cobraCmd.Println("│                                                             │")
		cobraCmd.Println("│  Action Required: Accept GitHub Invitation                  │")
		cobraCmd.Println("│                                                             │")
		if invErr.URL != "" {
			cobraCmd.Printf("│  %-59s │\n", invErr.URL)
			cobraCmd.Println("│                                                             │")
		}
		cobraCmd.Println("│  After accepting, clone the app with:                        │")
		cobraCmd.Printf("│  major app clone --app-id \"%s\"  │\n", createResp.ApplicationID)
		cobraCmd.Println("│                                                             │")
		cobraCmd.Println("╰─────────────────────────────────────────────────────────────╯")
		return nil // Exit cleanly, template pushed but clone pending user's invitation acceptance
	}

	if err != nil {
		return errors.WrapError("failed to ensure repository access", err)
	}

	// Select resources for the application
	cobraCmd.Println("\nSelecting resources for your application...")
	selectedResources, err := utils.SelectApplicationResources(cobraCmd, apiClient, orgID, createResp.ApplicationID)
	if err != nil {
		return errors.ErrorFailedToSelectResources
	}

	// Clone the repository (which now has template content)
	targetDir := filepath.Join(".", appName)
	cobraCmd.Printf("\nCloning repository to %s...\n", targetDir)
	_, gitErr := cloneRepository(createResp.CloneURLSSH, createResp.CloneURLHTTPS, targetDir)
	if gitErr != nil {
		return errors.WrapError("failed to clone repository", gitErr)
	}

	cobraCmd.Printf("✓ Application '%s' successfully created in ./%s\n", appName, appName)

	// If resources were selected, add them using major-client
	if len(selectedResources) > 0 {
		if err := utils.AddResourcesToProject(cobraCmd, targetDir, selectedResources, createResp.ApplicationID); err != nil {
			return errors.ErrorFailedToSelectResources
		}
	}

	// Generate .env file
	cobraCmd.Println("\nGenerating .env file...")
	envFilePath, envVars, err := generateEnvFile(targetDir)
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

	// Generate theme files
	cobraCmd.Println("Generating theme files...")
	if err := generateThemeFiles(targetDir); err != nil {
		cobraCmd.Printf("Warning: Failed to generate theme files: %v\n", err)
	} else {
		cobraCmd.Println("✓ Theme files generated")
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
		Foreground(lipgloss.Color("10")). // Green
		Bold(true)

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

	resourceCommand := commandStyle.Render("major resource manage")
	resourceDesc := descriptionStyle.Render("  Manage the resources your app is connected to")

	content := fmt.Sprintf("%s\n\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s",
		nextStepsTitle,
		cdInstruction,
		startCommand,
		startDesc,
		deployCommand,
		deployDesc,
		resourceCommand,
		resourceDesc,
	)

	box := boxStyle.Render(content)

	// Print everything
	cobraCmd.Println(successMsg)
	cobraCmd.Println(box)
}

