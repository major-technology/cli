package project

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	var versionFlag string
	var yes bool

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the latest compiled version of this project",
		Long:  `Deploys a compiled project version: agents are created, updated, or deleted to match the version's definitions. Deletions require confirmation.`,
		PreRunE: middleware.Compose(
			middleware.CheckLogin,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, versionFlag, yes)
		},
	}

	cmd.Flags().StringVar(&versionFlag, "version", "", "Commit hash (or unique prefix) of the version to deploy (default: latest compiled)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt (required for non-interactive deploys that delete)")

	return cmd
}

// resolveVersion picks the version to deploy from the project's version list.
func resolveVersion(versions []api.ProjectVersionItem, commitFlag string) (*api.ProjectVersionItem, error) {
	if commitFlag != "" {
		for i := range versions {
			if strings.HasPrefix(versions[i].CommitHash, commitFlag) {
				return &versions[i], nil
			}
		}
		return nil, fmt.Errorf("no version found for commit %q", commitFlag)
	}

	for i := range versions {
		if versions[i].CompileStatus == "compiled" {
			return &versions[i], nil
		}
	}

	return nil, fmt.Errorf("no compiled version available - push to main and check compile status with 'major project view'")
}

// renderPlan formats a deploy plan for the terminal.
func renderPlan(plan *api.GetProjectDeployPlanResponse) string {
	total := len(plan.Creates) + len(plan.Updates) + len(plan.Unchanged) + len(plan.Deletes)
	if total == 0 {
		return "No changes: the project has no agents in this version.\n"
	}

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	var b strings.Builder
	b.WriteString("Deploy plan:\n")

	for _, slug := range plan.Creates {
		b.WriteString(green.Render("  + create    ") + slug + "\n")
	}
	for _, slug := range plan.Updates {
		b.WriteString(yellow.Render("  ~ update    ") + slug + "\n")
	}
	for _, slug := range plan.Unchanged {
		b.WriteString(gray.Render("  = unchanged ") + slug + "\n")
	}
	for _, slug := range plan.Deletes {
		b.WriteString(red.Render("  - delete    ") + slug + "\n")
	}

	return b.String()
}

func runDeploy(cmd *cobra.Command, versionFlag string, yes bool) error {
	projectID, _, err := getProjectAndOrgID()
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()

	versionsResp, err := apiClient.ListProjectVersions(projectID)
	if err != nil {
		return err
	}

	version, err := resolveVersion(versionsResp.Versions, versionFlag)
	if err != nil {
		return errors.WrapError("failed to resolve version", err)
	}

	if version.CompileStatus != "compiled" {
		return fmt.Errorf("version %s failed to compile and cannot be deployed:\n%s", version.CommitHash[:12], version.CompileError)
	}

	cmd.Printf("Deploying version %s\n\n", version.CommitHash[:12])

	plan, err := apiClient.GetProjectDeployPlan(projectID, version.ID)
	if err != nil {
		return err
	}

	cmd.Print(renderPlan(plan))

	if len(plan.Deletes) > 0 && !yes {
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("This deploy DELETES %d agent(s). Deleted agents cannot be revived. Continue?", len(plan.Deletes))).
					Value(&confirm),
			),
		)

		if err := form.Run(); err != nil {
			return errors.WrapError("failed to confirm deploy", err)
		}

		if !confirm {
			return errors.ErrorOperationCancelled
		}
	}

	deployResp, err := apiClient.CreateProjectDeploy(projectID, version.ID)
	if err != nil {
		return err
	}

	cmd.Println()

	for _, artifact := range deployResp.Artifacts {
		switch artifact.Status {
		case "deployed", "unchanged", "deleted":
			cmd.Printf("✓ %s: %s\n", artifact.Slug, artifact.Status)
		default:
			cmd.Printf("✗ %s: %s %s\n", artifact.Slug, artifact.Status, artifact.Error)
		}
	}

	if deployResp.Status != "deployed" {
		return fmt.Errorf("deploy finished with status: %s", deployResp.Status)
	}

	cmd.Println("\n🎉 Deploy successful!")
	return nil
}
