package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// generateEnvCmd represents the generate_env command
var generateEnvCmd = &cobra.Command{
	Use:   "generate_env",
	Short: "Generate a .env file for the current application",
	Long:  `Generate a .env file at the root of the git repository with environment variables for the current application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runGenerateEnv(cmd))
	},
}

func runGenerateEnv(cmd *cobra.Command) error {
	envFilePath, numVars, err := generateEnvFile("")
	if err != nil {
		return err
	}

	cmd.Printf("Successfully generated .env file at: %s\n", envFilePath)
	cmd.Printf("Environment variables written: %d\n", numVars)

	return nil
}

// generateEnvFile generates a .env file for the application in the specified directory.
// If targetDir is empty, it uses the current git repository root.
// Returns the path to the generated file and the number of variables written.
func generateEnvFile(targetDir string) (string, int, error) {
	// Get the default organization ID from keyring
	orgID, _, err := token.GetDefaultOrg()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get default organization: %w\nPlease run 'major org select' to set a default organization", err)
	}

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

	// Get environment variables
	envVars, err := apiClient.GetApplicationEnv(orgID, applicationID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get environment variables: %w", err)
	}

	// Determine the target directory
	gitRoot := targetDir
	if gitRoot == "" {
		gitRoot, err = git.GetRepoRoot()
		if err != nil {
			return "", 0, fmt.Errorf("failed to get git repository root: %w", err)
		}
	}

	// Create .env file path
	envFilePath := filepath.Join(gitRoot, ".env")

	// Build the .env file content
	var envContent strings.Builder
	for key, value := range envVars {
		envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Write to .env file
	err = os.WriteFile(envFilePath, []byte(envContent.String()), 0644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to write .env file: %w", err)
	}

	return envFilePath, len(envVars), nil
}
