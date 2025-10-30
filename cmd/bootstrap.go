/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/huh"
	mjrToken "github.com/major-technology/cli/token"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CreateApplicationRequest represents the request body for POST /applications
type CreateApplicationRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	OrganizationID string `json:"organizationId"`
}

// CreateApplicationResponse represents the response from POST /applications
type CreateApplicationResponse struct {
	ApplicationID  string `json:"applicationId"`
	RepositoryName string `json:"repositoryName"`
	CloneURLSSH    string `json:"cloneUrlSsh"`
	CloneURLHTTPS  string `json:"cloneUrlHttps"`
}

const templateRepoURL = "https://github.com/major-technology/basic-template.git"

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Create a new application",
	Long:  `Bootstrap creates a new application with a GitHub repository and sets up the basic template.`,
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runBootstrap(cmd))
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}

func runBootstrap(cmd *cobra.Command) error {
	// Get token from keychain
	token, err := mjrToken.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated. Please run 'cli login' first: %w", err)
	}

	// Get default org from keychain
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return fmt.Errorf("no default organization set. Please run 'cli login' first: %w", err)
	}

	cmd.Printf("Creating application in organization: %s\n\n", orgName)

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

	cmd.Printf("\nCreating application '%s'...\n", appName)

	// Call POST /applications
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		return fmt.Errorf("api_url not configured")
	}

	createResp, err := createApplication(apiURL, token, appName, appDescription, orgID)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	cmd.Printf("✓ Application created with ID: %s\n", createResp.ApplicationID)
	cmd.Printf("✓ Repository: %s\n", createResp.RepositoryName)

	// Check if we have permissions to use SSH or HTTPS
	useSSH := false
	if canUseSSH() {
		cmd.Println("✓ SSH access detected")
		useSSH = true
	} else if createResp.CloneURLHTTPS != "" {
		cmd.Println("✓ Using HTTPS for git operations")
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

	cmd.Printf("\nCloning template repository...\n")

	// Clone the template repository
	if err := gitClone(templateRepoURL, tempDir); err != nil {
		return fmt.Errorf("failed to clone template repository: %w", err)
	}

	cmd.Println("✓ Template cloned")

	// Remove the existing remote origin
	if err := gitRemoveRemote(tempDir, "origin"); err != nil {
		return fmt.Errorf("failed to remove remote origin: %w", err)
	}

	cmd.Println("✓ Removed template remote")

	// Add the new remote
	if err := gitAddRemote(tempDir, "origin", cloneURL); err != nil {
		return fmt.Errorf("failed to add new remote: %w", err)
	}

	cmd.Printf("✓ Added new remote: %s\n", cloneURL)

	// Push to the new remote
	cmd.Println("\nPushing to new repository...")
	if err := gitPush(tempDir); err != nil {
		return fmt.Errorf("failed to push to new repository: %w", err)
	}

	cmd.Println("✓ Pushed to repository")

	// Move the repository to the current directory
	targetDir := filepath.Join(".", appName)
	if err := os.Rename(tempDir, targetDir); err != nil {
		return fmt.Errorf("failed to move repository: %w", err)
	}

	cmd.Printf("\n✓ Application '%s' successfully created in ./%s\n", appName, appName)
	cmd.Printf("  Clone URL: %s\n", cloneURL)

	return nil
}

// createApplication calls POST /applications
func createApplication(apiURL, token, name, description, orgID string) (*CreateApplicationResponse, error) {
	url := apiURL + "/applications"

	reqBody := CreateApplicationRequest{
		Name:           name,
		Description:    description,
		OrganizationID: orgID,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s", errResp.Message)
		}
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateApplicationResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	return &createResp, nil
}

// canUseSSH checks if SSH is available and configured for git
func canUseSSH() bool {
	// Check if ssh-agent is running and has keys
	cmd := exec.Command("ssh-add", "-l")
	err := cmd.Run()
	return err == nil
}

// gitClone clones a git repository
func gitClone(url, targetDir string) error {
	cmd := exec.Command("git", "clone", url, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitRemoveRemote removes a git remote
func gitRemoveRemote(repoDir, remoteName string) error {
	cmd := exec.Command("git", "remote", "remove", remoteName)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitAddRemote adds a git remote
func gitAddRemote(repoDir, remoteName, url string) error {
	cmd := exec.Command("git", "remote", "add", remoteName, url)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitPush pushes to the remote repository
func gitPush(repoDir string) error {
	cmd := exec.Command("git", "push", "--force", "-u", "origin", "main")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
