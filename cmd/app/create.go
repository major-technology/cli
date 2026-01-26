package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/constants"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Flag variables for non-interactive mode
var (
	flagAppName         string
	flagAppDescription  string
	flagTemplate        string
	flagCreateGithubUser string
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	Long: `Create a new application with a GitHub repository and sets up the basic template.

By default, this command runs interactively, prompting for application name, description, and template.
You can also provide these values via flags for non-interactive usage:

  major app create --name "my-app" --description "My application" --template "Vite"

Available templates: Vite, NextJS

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
	createCmd.Flags().StringVar(&flagTemplate, "template", "", "Template name: 'Vite' or 'NextJS' (skips interactive prompt)")
	createCmd.Flags().StringVar(&flagCreateGithubUser, "github-user", "", "GitHub username for repository access (for non-interactive mode)")
}

func runCreate(cobraCmd *cobra.Command) error {
	// Get default org from keychain
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return errors.ErrorNoOrganizationSelected
	}

	cobraCmd.Printf("Creating application in organization: %s\n\n", orgName)

	// Use flag values if provided, otherwise prompt interactively
	appName := flagAppName
	appDescription := flagAppDescription

	// Check if we need to prompt for any values
	needsPrompt := appName == "" || appDescription == ""

	if needsPrompt {
		// Build form fields only for missing values
		var formGroups []huh.Field

		if appName == "" {
			formGroups = append(formGroups,
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
			formGroups = append(formGroups,
				huh.NewText().
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

		form := huh.NewForm(huh.NewGroup(formGroups...))

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

	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Fetch and select template
	_, templateName, templateID, err := selectTemplate(cobraCmd, apiClient)
	if err != nil {
		return errors.WrapError("failed to select template", err)
	}

	cobraCmd.Printf("\nCreating application '%s'...\n", appName)

	// Call POST /applications (token will be fetched automatically)
	createResp, err := apiClient.CreateApplication(appName, appDescription, orgID)
	if err != nil {
		return err
	}

	cobraCmd.Printf("✓ Application created with ID: %s\n", createResp.ApplicationID)
	cobraCmd.Printf("✓ Repository: %s\n", createResp.RepositoryName)

	_, err = apiClient.SetApplicationTemplate(createResp.ApplicationID, templateID)
	if err != nil {
		return err
	}

	// Push template files to repository using backend (GitHub App credentials)
	// This bypasses user's SSH access, so template is pushed even if invitation is pending
	cobraCmd.Println("Pushing template to repository...")
	pushResp, err := apiClient.PushTemplate(createResp.ApplicationID, templateID)
	if err != nil {
		return errors.WrapError("failed to push template to repository", err)
	}

	if !pushResp.Success {
		return errors.WrapError("failed to push template to repository", fmt.Errorf("%s", pushResp.ErrorMsg))
	}

	cobraCmd.Printf("✓ Template pushed (%d files)\n", pushResp.FilesCount)

	// Ensure repository access before cloning
	// Use non-interactive mode if all required flags were provided
	isNonInteractive := flagAppName != "" && flagAppDescription != "" && flagTemplate != ""
	opts := utils.EnsureRepositoryAccessOptions{
		NonInteractive: isNonInteractive,
		GithubUsername: flagCreateGithubUser,
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
		if err := utils.AddResourcesToProject(cobraCmd, targetDir, selectedResources, createResp.ApplicationID, templateName); err != nil {
			return errors.ErrorFailedToSelectResources
		}
	}

	// Generate .env file
	cobraCmd.Println("\nGenerating .env file...")
	envFilePath, _, err := generateEnvFile(targetDir)
	if err != nil {
		cobraCmd.Printf("Warning: Failed to generate .env file: %v\n", err)
	} else {
		cobraCmd.Printf("✓ Generated .env file at: %s\n", envFilePath)
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

// selectTemplate prompts the user to select a template for the application
// Returns the template URL, name, and ID
func selectTemplate(cobraCmd *cobra.Command, apiClient *api.Client) (string, constants.TemplateName, string, error) {
	// Fetch available templates
	templatesResp, err := apiClient.GetTemplates()
	if err != nil {
		return "", "", "", err
	}

	// Prioritize the recommended template (this is the vite template rn)
	recommendedID := "962add46-30fb-48b6-94a6-7b967cdf0d35"
	var orderedTemplates []*api.TemplateItem

	for _, t := range templatesResp.Templates {
		if t.ID == recommendedID {
			orderedTemplates = append([]*api.TemplateItem{t}, orderedTemplates...)
		} else {
			orderedTemplates = append(orderedTemplates, t)
		}
	}

	// Check if there are any templates available
	if len(orderedTemplates) == 0 {
		return "", "", "", errors.ErrorNoTemplatesAvailable
	}

	// If template flag is provided, find and use that template
	if flagTemplate != "" {
		for _, template := range templatesResp.Templates {
			if strings.EqualFold(string(template.Name), flagTemplate) {
				cobraCmd.Printf("Using template: %s\n", template.Name)
				return template.TemplateURL, template.Name, template.ID, nil
			}
		}
		// Template not found - list available templates
		var availableTemplates []string
		for _, t := range templatesResp.Templates {
			availableTemplates = append(availableTemplates, string(t.Name))
		}
		return "", "", "", fmt.Errorf("template '%s' not found. Available templates: %s", flagTemplate, strings.Join(availableTemplates, ", "))
	}

	// If only one template, use it automatically
	if len(orderedTemplates) == 1 {
		template := orderedTemplates[0]
		cobraCmd.Printf("Using template: %s\n", template.Name)
		return template.TemplateURL, template.Name, template.ID, nil
	}

	// Create options for the select (add display suffix for recommended template)
	options := make([]huh.Option[string], len(orderedTemplates))
	for i, template := range orderedTemplates {
		displayName := string(template.Name)
		if template.ID == recommendedID {
			displayName += " (recommended)"
		}
		options[i] = huh.NewOption(displayName, template.TemplateURL)
	}

	// Prompt user to select a template
	var selectedTemplateURL string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a template for your application").
				Description("Choose which template to use as a starting point").
				Options(options...).
				Value(&selectedTemplateURL),
		),
	)

	if err := form.Run(); err != nil {
		return "", "", "", errors.WrapError("failed to select template", err)
	}

	// Find the template name and ID for the selected URL
	var selectedTemplateName constants.TemplateName
	var selectedTemplateID string
	for _, template := range templatesResp.Templates {
		if template.TemplateURL == selectedTemplateURL {
			selectedTemplateName = template.Name
			selectedTemplateID = template.ID
			break
		}
	}

	return selectedTemplateURL, selectedTemplateName, selectedTemplateID, nil
}
