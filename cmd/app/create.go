package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

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

	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Fetch and select template
	cobraCmd.Println("\nFetching available templates...")
	templateURL, templateName, err := selectTemplate(cobraCmd, apiClient)
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
	selectedResources, err := selectApplicationResources(cobraCmd, orgID, createResp.ApplicationID)
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
		if err := addResourcesToViteProject(cobraCmd, targetDir, selectedResources, createResp.ApplicationID); err != nil {
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

// addResourcesToViteProject adds selected resources to a Vite project using pnpm clients:add
func addResourcesToViteProject(cobraCmd *cobra.Command, projectDir string, resources []api.ResourceItem, applicationID string) error {
	// First, install dependencies to make major-client available
	cobraCmd.Println("  Installing dependencies...")
	installCmd := exec.Command("pnpm", "install")
	installCmd.Dir = projectDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	successCount := 0
	for _, resource := range resources {
		// Convert resource name to a valid client name (kebab-case)
		// The major-client tool will convert it to camelCase for the actual client
		clientName := resource.Name

		cobraCmd.Printf("  Adding resource: %s (%s)...\n", resource.Name, resource.Type)

		// Run: pnpm clients:add <resource_id> <name> <type> <description> <application_id>
		cmd := exec.Command("pnpm", "clients:add", resource.ID, clientName, resource.Type, resource.Description, applicationID)
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			cobraCmd.Printf("  ⚠ Failed to add resource %s: %v\n", resource.Name, err)
			continue
		}

		successCount++
	}

	if successCount > 0 {
		cobraCmd.Printf("✓ Successfully added %d/%d resource(s) to the project\n", successCount, len(resources))
	}

	if successCount < len(resources) {
		return fmt.Errorf("failed to add %d resource(s)", len(resources)-successCount)
	}

	return nil
}

// selectApplicationResources prompts the user to select resources for the application
// Returns the selected resources with their full details
func selectApplicationResources(cobraCmd *cobra.Command, orgID, appID string) ([]api.ResourceItem, error) {
	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Fetch available resources
	resourcesResp, err := apiClient.GetResources(orgID)
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return nil, err
	}

	// Check if there are any resources available
	if len(resourcesResp.Resources) == 0 {
		cobraCmd.Println("No resources available in this organization.")
		return nil, nil
	}

	// Create options for the multiselect
	options := make([]huh.Option[string], len(resourcesResp.Resources))
	for i, resource := range resourcesResp.Resources {
		// Format: "Name - Description"
		label := resource.Name
		if resource.Description != "" {
			label = fmt.Sprintf("%s - %s", resource.Name, resource.Description)
		}
		options[i] = huh.NewOption(label, resource.ID)
	}

	// Create custom keymap where 'n' submits instead of enter
	customKeyMap := huh.NewDefaultKeyMap()
	customKeyMap.MultiSelect.Toggle = key.NewBinding(
		key.WithKeys(" ", "enter"),
		key.WithHelp("space/enter", "toggle"),
	)
	customKeyMap.MultiSelect.Submit = key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "continue"),
	)
	// Disable the default next/prev behavior on enter
	customKeyMap.MultiSelect.Next = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	)

	// Prompt user to select resources
	var selectedResourceIDs []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select resources for your application").
				Description("Use space/enter to select, 'n' to continue").
				Options(options...).
				Value(&selectedResourceIDs),
		),
	).WithKeyMap(customKeyMap)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("failed to collect resource selection: %w", err)
	}

	// If no resources selected, just return
	if len(selectedResourceIDs) == 0 {
		cobraCmd.Println("No resources selected.")
		return nil, nil
	}

	// Save the selected resources
	cobraCmd.Printf("Saving %d selected resource(s)...\n", len(selectedResourceIDs))
	_, err = apiClient.SaveApplicationResources(orgID, appID, selectedResourceIDs)
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return nil, err
	}

	cobraCmd.Printf("✓ Resources configured successfully\n")

	// Build and return the list of selected resources with full details
	var selectedResources []api.ResourceItem
	for _, selectedID := range selectedResourceIDs {
		for _, resource := range resourcesResp.Resources {
			if resource.ID == selectedID {
				selectedResources = append(selectedResources, resource)
				break
			}
		}
	}

	return selectedResources, nil
}

// selectTemplate prompts the user to select a template for the application
// Returns the template URL and name
func selectTemplate(cobraCmd *cobra.Command, apiClient *api.Client) (string, string, error) {
	// Fetch available templates
	templatesResp, err := apiClient.GetTemplates()
	if ok := api.CheckErr(cobraCmd, err); !ok {
		return "", "", err
	}

	// Check if there are any templates available
	if len(templatesResp.Templates) == 0 {
		return "", "", fmt.Errorf("no templates available")
	}

	// If only one template, use it automatically
	if len(templatesResp.Templates) == 1 {
		template := templatesResp.Templates[0]
		cobraCmd.Printf("Using template: %s\n", template.Name)
		return template.TemplateURL, template.Name, nil
	}

	// Create options for the select
	options := make([]huh.Option[string], len(templatesResp.Templates))
	for i, template := range templatesResp.Templates {
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
		return "", "", fmt.Errorf("failed to select template: %w", err)
	}

	// Find the template name for the selected URL
	var selectedTemplateName string
	for _, template := range templatesResp.Templates {
		if template.TemplateURL == selectedTemplateURL {
			selectedTemplateName = template.Name
			break
		}
	}

	return selectedTemplateURL, selectedTemplateName, nil
}
