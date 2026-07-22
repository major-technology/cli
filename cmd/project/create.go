package project

import (
	"path/filepath"

	"github.com/major-technology/cli/clients/git"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Long:  `Creates a new project: a GitHub repository from the project template, registered with the platform.`,
		Args:  cobra.ExactArgs(1),
		PreRunE: middleware.Compose(
			middleware.CheckLogin,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, args[0], description)
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Project description")

	return cmd
}

func runCreate(cmd *cobra.Command, name, description string) error {
	orgID, orgName, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return errors.ErrorNoOrganizationSelected
	}

	cmd.Printf("Creating project in organization: %s\n\n", orgName)

	apiClient := singletons.GetAPIClient()

	resp, err := apiClient.CreateProject(name, description, orgID)
	if err != nil {
		return err
	}

	cmd.Printf("✓ Project created with ID: %s\n", resp.ProjectID)
	cmd.Printf("✓ Repository: %s\n", resp.RepositoryName)

	// Grant the user's GitHub account access to the new repository.
	if githubUser, _ := git.GetCurrentGithubUser(); githubUser != "" {
		if _, err := apiClient.AddProjectGithubCollaborators(resp.ProjectID, githubUser); err != nil {
			cmd.Printf("Warning: failed to add you as a repository collaborator: %v\n", err)
			cmd.Printf("Clone manually once you have access: %s\n", resp.CloneURLHTTPS)
			return nil
		}
	}

	targetDir := filepath.Join(".", name)
	cmd.Printf("\nCloning repository to %s...\n", targetDir)

	cloneURL := resp.CloneURLHTTPS
	if utils.CanUseSSH() && resp.CloneURLSSH != "" {
		cloneURL = resp.CloneURLSSH
	}

	if err := git.Clone(cloneURL, targetDir); err != nil {
		cmd.Printf("Warning: clone failed (GitHub permissions may still be propagating): %v\n", err)
		cmd.Printf("Clone manually with: git clone %s\n", cloneURL)
		return nil
	}

	cmd.Printf("\n✓ Project '%s' created in ./%s\n", name, name)
	cmd.Println("\nNext steps:")
	cmd.Println("  1. Add agents under src/agents/<name>/agent.json")
	cmd.Println("  2. major project validate")
	cmd.Println("  3. git push (compiles automatically)")
	cmd.Println("  4. major project deploy")

	return nil
}
