package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/cmd/user"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// linkCmd represents the link command (hidden - used by install script)
var linkCmd = &cobra.Command{
	Use:    "link [application-id]",
	Short:  "Link and run an application locally",
	Long:   `Links an application by ID - logs in if needed, clones the repository, and starts the development server.`,
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLink(cmd, args[0])
	},
}

func init() {
	Cmd.AddCommand(linkCmd)
}

func runLink(cmd *cobra.Command, applicationID string) error {
	// Step 1: Check if user is logged in, login if not
	_, err := mjrToken.GetToken()
	if err != nil {
		if err := user.RunLoginForLink(cmd); err != nil {
			return errors.WrapError("failed to login", err)
		}
	}

	// Step 2: Fetch application info using the application ID
	cmd.Println("Fetching application info...")
	apiClient := singletons.GetAPIClient()

	appInfo, err := apiClient.GetApplicationForLink(applicationID)
	if err != nil {
		return errors.WrapError("failed to get application info", err)
	}

	cmd.Printf("Found application: %s\n", appInfo.Name)

	if err := mjrToken.StoreDefaultOrg(appInfo.OrganizationID, appInfo.OrganizationName); err != nil {
		return errors.WrapError("failed to store default organization", err)
	}

	// Step 3: Clone the repository or ensure existing directory is properly set up
	workingDir := sanitizeDirName(appInfo.Name)

	// Ensure the directory is a properly configured git repository
	gitErr := ensureGitRepository(cmd, workingDir, appInfo.CloneURLSSH, appInfo.CloneURLHTTPS)
	if gitErr != nil {
		if isGitAuthError(gitErr) {
			// Ensure repository access
			if err := utils.EnsureRepositoryAccess(cmd, applicationID, appInfo.CloneURLSSH, appInfo.CloneURLHTTPS); err != nil {
				return errors.WrapError("failed to ensure repository access", err)
			}
			// Retry with retries
			gitErr = ensureGitRepositoryWithRetries(cmd, workingDir, appInfo.CloneURLSSH, appInfo.CloneURLHTTPS)
			if gitErr != nil {
				return errors.ErrorGitRepositoryAccessFailed
			}
		} else {
			return errors.ErrorGitCloneFailed
		}
	}

	cmd.Println("✓ Repository ready")

	// Step 4: Generate .env file
	cmd.Println("Generating .env file...")
	envFilePath, envVars, err := generateEnvFile(workingDir)
	if err != nil {
		return errors.WrapError("failed to generate .env file", err)
	}
	cmd.Printf("✓ Generated .env file at: %s\n", envFilePath)

	// Generate .mcp.json for Claude Code
	if _, err := utils.GenerateMcpConfig(workingDir, envVars); err != nil {
		cmd.Printf("Warning: Failed to generate .mcp.json: %v\n", err)
	} else {
		cmd.Println("✓ Generated .mcp.json for Claude Code")
	}

	// Step 5: Print success and run start
	printLinkSuccessMessage(cmd, workingDir, appInfo.Name)

	// Step 6: Run pnpm install and pnpm dev in the target directory
	return RunStartInDir(cmd, workingDir)
}

func printLinkSuccessMessage(cmd *cobra.Command, dir, appName string) {
	// Define styles
	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")). // Green
		MarginTop(1).
		MarginBottom(1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")). // Blue
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	pathStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")) // Cyan

	// Build the message
	successMsg := successStyle.Render(fmt.Sprintf("🎉 Successfully linked %s!", appName))

	content := fmt.Sprintf("Application cloned to: %s", pathStyle.Render(dir))

	box := boxStyle.Render(content)

	// Print everything
	cmd.Println(successMsg)
	cmd.Println(box)
}
