package app

import (
	"fmt"
	"os"
	"os/exec"
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
	// Get the default organization ID from keyring
	orgID, _, err := token.GetDefaultOrg()
	if err != nil {
		return fmt.Errorf("failed to get default organization: %w\nPlease run 'major org select' to set a default organization", err)
	}

	// Get the git remote URL from the current directory
	remoteURL, err := git.GetRemoteURL()
	if err != nil {
		return fmt.Errorf("failed to get git remote: %w", err)
	}

	if remoteURL == "" {
		return fmt.Errorf("no git remote found in current directory")
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse git remote URL: %w", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	// Get environment variables
	envVars, err := apiClient.GetApplicationEnv(orgID, appResp.ApplicationID)
	if err != nil {
		return fmt.Errorf("failed to get environment variables: %w", err)
	}

	// Get git repository root
	gitRoot, err := getGitRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get git repository root: %w", err)
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
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	cmd.Printf("Successfully generated .env file at: %s\n", envFilePath)
	cmd.Printf("Environment variables written: %d\n", len(envVars))

	return nil
}

// getGitRepoRoot returns the root directory of the git repository
func getGitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

