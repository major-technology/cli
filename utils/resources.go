package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// LocalResource represents a resource stored in resources.json
type LocalResource struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	ApplicationID string `json:"applicationId"`
}

// ReadLocalResources reads the resources.json file from the project directory
func ReadLocalResources(projectDir string) ([]LocalResource, error) {
	resourcesPath := filepath.Join(projectDir, "resources.json")

	// If file doesn't exist, return empty list (not an error)
	if _, err := os.Stat(resourcesPath); os.IsNotExist(err) {
		return []LocalResource{}, nil
	}

	data, err := os.ReadFile(resourcesPath)
	if err != nil {
		return nil, errors.WrapError("failed to read resources.json", err)
	}

	var resources []LocalResource
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil, errors.WrapError("failed to parse resources.json", err)
	}

	return resources, nil
}

// SelectApplicationResources prompts the user to select resources for the application
// Returns the selected resources with their full details
func SelectApplicationResources(cmd *cobra.Command, apiClient *api.Client, orgID, appID string) ([]api.ResourceItem, error) {
	// Fetch available resources
	resourcesResp, err := apiClient.GetResources(orgID)
	if err != nil {
		return nil, err
	}

	// Check if there are any resources available
	if len(resourcesResp.Resources) == 0 {
		return nil, nil
	}

	// Try to read existing resources from resources.json
	existingResources, err := ReadLocalResources(".")
	if err != nil {
		cmd.Printf("Warning: Could not read existing resources: %v\n", err)
		existingResources = []LocalResource{}
	}

	// Create a set of existing resource IDs for quick lookup
	existingIDs := make(map[string]bool)
	for _, res := range existingResources {
		existingIDs[res.ID] = true
	}

	// Pre-select existing resources
	var selectedResourceIDs []string
	for _, res := range existingResources {
		selectedResourceIDs = append(selectedResourceIDs, res.ID)
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

	// Show count of existing selections if any
	title := "Select resources for your application"
	if len(existingResources) > 0 {
		title = fmt.Sprintf("Select resources for your application (%d currently selected)", len(existingResources))
	}

	// Prompt user to select resources
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Description("Use space/enter to select, 'n' to continue").
				Options(options...).
				Value(&selectedResourceIDs),
		),
	).WithKeyMap(customKeyMap)

	if err := form.Run(); err != nil {
		return nil, errors.WrapError("failed to collect resource selection", err)
	}

	_, err = apiClient.SaveApplicationResources(orgID, appID, selectedResourceIDs)
	if err != nil {
		return nil, err
	}

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

// AddResourcesToProject adds selected resources to a project using pnpm resource:add
// It handles differential updates: removes resources that are no longer selected and adds new ones
func AddResourcesToProject(cmd *cobra.Command, projectDir string, resources []api.ResourceItem, applicationID string) error {
	// Read existing resources
	existingResources, err := ReadLocalResources(projectDir)
	if err != nil {
		cmd.Printf("Warning: Could not read existing resources: %v\n", err)
		existingResources = []LocalResource{}
	}

	// Build maps for comparison
	newResourceMap := make(map[string]api.ResourceItem)
	for _, res := range resources {
		newResourceMap[res.ID] = res
	}

	existingResourceMap := make(map[string]LocalResource)
	for _, res := range existingResources {
		existingResourceMap[res.ID] = res
	}

	// Find resources to remove (in old but not in new)
	var resourcesToRemove []LocalResource
	for _, existing := range existingResources {
		if _, found := newResourceMap[existing.ID]; !found {
			resourcesToRemove = append(resourcesToRemove, existing)
		}
	}

	// Find resources to add (in new but not in old)
	var resourcesToAdd []api.ResourceItem
	for _, newRes := range resources {
		if _, found := existingResourceMap[newRes.ID]; !found {
			resourcesToAdd = append(resourcesToAdd, newRes)
		}
	}

	// If nothing to change, return early
	if len(resourcesToRemove) == 0 && len(resourcesToAdd) == 0 {
		cmd.Println("No changes to resources.")
		return nil
	}

	// First, install dependencies to make major-client available
	cmd.Println("  Installing dependencies...")
	installCmd := exec.Command("pnpm", "install")
	installCmd.Dir = projectDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		return errors.WrapError("failed to install dependencies", err)
	}

	prefix := "resource"

	// Remove old resources
	removeSuccessCount := 0
	for _, resource := range resourcesToRemove {
		cmd.Printf("  Removing resource: %s (%s)...\n", resource.Name, resource.Type)

		// Run: pnpm clients:remove <name>
		pnpmCmd := exec.Command("pnpm", prefix+":remove", resource.Name)
		pnpmCmd.Dir = projectDir
		pnpmCmd.Stdout = os.Stdout
		pnpmCmd.Stderr = os.Stderr

		if err := pnpmCmd.Run(); err != nil {
			cmd.Printf("  ⚠ Failed to remove resource %s: %v\n", resource.Name, err)
			continue
		}

		removeSuccessCount++
	}

	// Add new resources
	addSuccessCount := 0
	for _, resource := range resourcesToAdd {
		// Convert resource name to a valid client name
		// The major-client tool will convert it to camelCase for the actual client
		clientName := resource.Name

		cmd.Printf("  Adding resource: %s (%s)...\n", resource.Name, resource.Type)

		// Run: pnpm clients:add <resource_id> <name> <type> <description> <application_id>
		pnpmCmd := exec.Command("pnpm", prefix+":add", resource.ID, clientName, resource.Type, resource.Description, applicationID)
		pnpmCmd.Dir = projectDir
		pnpmCmd.Stdout = os.Stdout
		pnpmCmd.Stderr = os.Stderr

		if err := pnpmCmd.Run(); err != nil {
			cmd.Printf("  ⚠ Failed to add resource %s: %v\n", resource.Name, err)
			continue
		}

		addSuccessCount++
	}

	// Report results
	if removeSuccessCount > 0 {
		cmd.Printf("✓ Successfully removed %d/%d resource(s)\n", removeSuccessCount, len(resourcesToRemove))
	}
	if addSuccessCount > 0 {
		cmd.Printf("✓ Successfully added %d/%d resource(s)\n", addSuccessCount, len(resourcesToAdd))
	}

	totalErrors := (len(resourcesToRemove) - removeSuccessCount) + (len(resourcesToAdd) - addSuccessCount)
	if totalErrors > 0 {
		return fmt.Errorf("failed to process %d resource(s)", totalErrors)
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
		return "", 0, errors.WrapError("failed to get application resources", err)
	}

	// Determine the target directory
	gitRoot := targetDir
	if gitRoot == "" {
		gitRoot, err = git.GetRepoRoot()
		if err != nil {
			return "", 0, errors.WrapError("failed to get git repository root", err)
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
		return "", 0, errors.WrapError("failed to write RESOURCES.md file", err)
	}

	return resourcesFilePath, len(resourcesResp.Resources), nil
}
