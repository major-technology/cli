package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	Long:  `Create a new application with a GitHub repository and sets up the basic template.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
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

	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Fetch and select template
	cobraCmd.Println("\nFetching available templates...")
	templateURL, templateName, templateID, err := selectTemplate(cobraCmd, apiClient)
	if err != nil {
		return fmt.Errorf("failed to select template: %w", err)
	}

	cobraCmd.Printf("\nCreating application '%s'...\n", appName)

	// Call POST /applications (token will be fetched automatically)
	createResp, err := apiClient.CreateApplication(appName, appDescription, orgID)
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return err
	}

	cobraCmd.Printf("✓ Application created with ID: %s\n", createResp.ApplicationID)
	cobraCmd.Printf("✓ Repository: %s\n", createResp.RepositoryName)

	_, err = apiClient.SetApplicationTemplate(createResp.ApplicationID, templateID)
	if ok := api.CheckErr(cobraCmd, err); !ok {
		// Don't fail the entire create if template setting fails
	}

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
	if err := git.Clone(templateURL, tempDir); err != nil {
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

	// Select resources for the application
	cobraCmd.Println("\nSelecting resources for your application...")
	selectedResources, err := utils.SelectApplicationResources(cobraCmd, apiClient, orgID, createResp.ApplicationID)
	if err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red
		cobraCmd.Println(errorStyle.Render("Failed to configure resources. Please run 'major app resources' to configure them later."))
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

	// If Vite template and resources were selected, add them using major-client
	if templateName == "Vite" && len(selectedResources) > 0 {
		cobraCmd.Println("\nAdding resources to Vite project...")
		if err := utils.AddResourcesToViteProject(cobraCmd, targetDir, selectedResources, createResp.ApplicationID); err != nil {
			cobraCmd.Printf("Warning: Failed to add resources to project: %v\n", err)
			cobraCmd.Println("You can manually add them later using 'pnpm clients:add'")
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

// selectTemplate prompts the user to select a template for the application
// Returns the template URL, name, and ID
func selectTemplate(cobraCmd *cobra.Command, apiClient *api.Client) (string, string, string, error) {
	// Fetch available templates
	templatesResp, err := apiClient.GetTemplates()
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return "", "", "", err
	}

	// Prioritize the recommended template (this is the vite template rn)
	recommendedID := "962add46-30fb-48b6-94a6-7b967cdf0d35"
	var orderedTemplates []*api.TemplateItem

	for _, t := range templatesResp.Templates {
		if t.ID == recommendedID {
			t.Name = t.Name + " (recommended)"
			orderedTemplates = append([]*api.TemplateItem{t}, orderedTemplates...)
		} else {
			orderedTemplates = append(orderedTemplates, t)
		}
	}

	// Check if there are any templates available
	if len(orderedTemplates) == 0 {
		return "", "", "", errors.New("no templates available")
	}

	// If only one template, use it automatically
	if len(orderedTemplates) == 1 {
		template := orderedTemplates[0]
		cobraCmd.Printf("Using template: %s\n", template.Name)
		return template.TemplateURL, template.Name, template.ID, nil
	}

	// Create options for the select
	options := make([]huh.Option[string], len(orderedTemplates))
	for i, template := range orderedTemplates {
		options[i] = huh.NewOption(template.Name, template.TemplateURL)
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
		return "", "", "", fmt.Errorf("failed to select template: %w", err)
	}

	// Find the template name and ID for the selected URL
	var selectedTemplateName, selectedTemplateID string
	for _, template := range orderedTemplates {
		if template.TemplateURL == selectedTemplateURL {
			selectedTemplateName = template.Name
			selectedTemplateID = template.ID
			break
		}
	}

	return selectedTemplateURL, selectedTemplateName, selectedTemplateID, nil
}
