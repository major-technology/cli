package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// generateResourcesCmd represents the generate_resources command
var generateResourcesCmd = &cobra.Command{
	Use:   "generate_resources",
	Short: "Generate a RESOURCES.md file for the current application",
	Long:  `Generate a RESOURCES.md file at the root of the git repository with resources for the current application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runGenerateResources(cmd))
	},
}

func runGenerateResources(cmd *cobra.Command) error {
	resourcesFilePath, _, err := generateResourcesFile("")
	if err != nil {
		return err
	}

	cmd.Printf("Successfully generated RESOURCES.md file at: %s\n", resourcesFilePath)

	return nil
}

// generateResourcesFile generates a RESOURCES.md file for the application in the specified directory.
// If targetDir is empty, it uses the current git repository root.
// Returns the path to the generated file and the number of resources written.
func generateResourcesFile(targetDir string) (string, int, error) {
	// Get application ID from the specified directory (or current if empty)
	applicationID, err := getApplicationIDFromDir(targetDir)
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
