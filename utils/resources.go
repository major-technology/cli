package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// SelectApplicationResources prompts the user to select resources for the application
// Returns the selected resources with their full details
func SelectApplicationResources(cmd *cobra.Command, apiClient *api.Client, orgID, appID string) ([]api.ResourceItem, error) {
	// Fetch available resources
	resourcesResp, err := apiClient.GetResources(orgID)
	if ok := api.CheckErr(cmd, err); !ok {
		return nil, err
	}

	// Check if there are any resources available
	if len(resourcesResp.Resources) == 0 {
		cmd.Println("No resources available in this organization.")
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
		cmd.Println("No resources selected.")
		return nil, nil
	}

	// Save the selected resources
	cmd.Printf("Saving %d selected resource(s)...\n", len(selectedResourceIDs))
	_, err = apiClient.SaveApplicationResources(orgID, appID, selectedResourceIDs)
	if ok := api.CheckErr(cmd, err); !ok {
		return nil, err
	}

	cmd.Printf("✓ Resources configured successfully\n")

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

// AddResourcesToViteProject adds selected resources to a Vite project using pnpm clients:add
func AddResourcesToViteProject(cmd *cobra.Command, projectDir string, resources []api.ResourceItem, applicationID string) error {
	// First, install dependencies to make major-client available
	cmd.Println("  Installing dependencies...")
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

		cmd.Printf("  Adding resource: %s (%s)...\n", resource.Name, resource.Type)

		// Run: pnpm clients:add <resource_id> <name> <type> <description> <application_id>
		pnpmCmd := exec.Command("pnpm", "clients:add", resource.ID, clientName, resource.Type, resource.Description, applicationID)
		pnpmCmd.Dir = projectDir
		pnpmCmd.Stdout = os.Stdout
		pnpmCmd.Stderr = os.Stderr

		if err := pnpmCmd.Run(); err != nil {
			cmd.Printf("  ⚠ Failed to add resource %s: %v\n", resource.Name, err)
			continue
		}

		successCount++
	}

	if successCount > 0 {
		cmd.Printf("✓ Successfully added %d/%d resource(s) to the project\n", successCount, len(resources))
	}

	if successCount < len(resources) {
		return fmt.Errorf("failed to add %d resource(s)", len(resources)-successCount)
	}

	return nil
}

// GenerateResourcesFile generates a RESOURCES.md file for the application in the specified directory.
// If targetDir is empty, it uses the current git repository root.
// Returns the path to the generated file and the number of resources written.
func GenerateResourcesFile(targetDir string) (string, int, error) {
	// Get application ID from the specified directory (or current if empty)
	applicationID, _, err := GetApplicationIDFromDir(targetDir)
	if err != nil {
		return "", 0, err
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return "", 0, fmt.Errorf("API client not initialized")
	}

	// Get application resources
	resourcesResp, err := apiClient.GetApplicationResources(applicationID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get application resources: %w", err)
	}

	// Determine the target directory
	gitRoot := targetDir
	if gitRoot == "" {
		gitRoot, err = git.GetRepoRoot()
		if err != nil {
			return "", 0, fmt.Errorf("failed to get git repository root: %w", err)
		}
	}

	// Create RESOURCES.md file path
	resourcesFilePath := filepath.Join(gitRoot, "RESOURCES.md")

	// Build the RESOURCES.md file content
	var content strings.Builder
	content.WriteString("## Resources\n")

	for i, resource := range resourcesResp.Resources {
		if i > 0 {
			content.WriteString("\n------\n\n")
		} else {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("ID: %s\n", resource.ID))
		content.WriteString(fmt.Sprintf("Name: %s\n", resource.Name))
		content.WriteString(fmt.Sprintf("Description: %s\n", resource.Description))
	}

	// Write to RESOURCES.md file
	err = os.WriteFile(resourcesFilePath, []byte(content.String()), 0644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to write RESOURCES.md file: %w", err)
	}

	return resourcesFilePath, len(resourcesResp.Resources), nil
}
