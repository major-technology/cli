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
// An exact commit-hash match is always a direct hit. A prefix match must be
// unique: an ambiguous prefix (matching more than one version) is refused
// with the candidates listed, rather than silently picking one on a command
// that can delete agents.
func resolveVersion(versions []api.ProjectVersionItem, commitFlag string) (*api.ProjectVersionItem, error) {
	if commitFlag != "" {
		for i := range versions {
			if versions[i].CommitHash == commitFlag {
				return &versions[i], nil
			}
		}

		var matches []*api.ProjectVersionItem
		for i := range versions {
			if strings.HasPrefix(versions[i].CommitHash, commitFlag) {
				matches = append(matches, &versions[i])
			}
		}

		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("no version found for commit %q", commitFlag)
		case 1:
			return matches[0], nil
		default:
			var b strings.Builder
			fmt.Fprintf(&b, "commit %q matches %d versions, use a longer prefix to disambiguate:\n", commitFlag, len(matches))
			for _, v := range matches {
				fmt.Fprintf(&b, "  %s (%s)\n", shortHash(v.CommitHash), v.CreatedAt)
			}
			return nil, fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
		}
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
	if total == 0 && len(plan.Warnings) == 0 {
		return "No changes: the project has no agents in this version.\n"
	}

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	orange := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

	var b strings.Builder

	if total > 0 {
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
	}

	if len(plan.Warnings) > 0 {
		if total > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Warnings:\n")
		for _, warning := range plan.Warnings {
			b.WriteString(orange.Render("  ⚠ ") + warning + "\n")
		}
	}

	return b.String()
}

func runDeploy(cmd *cobra.Command, versionFlag string, yes bool) error {
	projectID, orgID, err := getProjectAndOrgID()
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()

	versionsResp, err := apiClient.ListProjectVersions(projectID, orgID)
	if err != nil {
		return err
	}

	version, err := resolveVersion(versionsResp.Versions, versionFlag)
	if err != nil {
		return errors.WrapError("failed to resolve version", err)
	}

	if version.CompileStatus != "compiled" {
		return fmt.Errorf("version %s failed to compile and cannot be deployed:\n%s", shortHash(version.CommitHash), version.CompileError)
	}

	cmd.Printf("Deploying version %s\n\n", shortHash(version.CommitHash))

	plan, err := apiClient.GetProjectDeployPlan(projectID, orgID, version.ID)
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

	deployResp, err := apiClient.CreateProjectDeploy(projectID, orgID, version.ID)
	if err != nil {
		return err
	}

	cmd.Println()

	for _, artifact := range deployResp.Artifacts {
		switch artifact.Status {
		case "deployed", "unchanged", "deleted":
			cmd.Printf("✓ %s: %s\n", artifact.Slug, artifact.Status)
		default:
			if artifact.Error != "" {
				cmd.Printf("✗ %s: %s %s\n", artifact.Slug, artifact.Status, artifact.Error)
			} else {
				cmd.Printf("✗ %s: %s\n", artifact.Slug, artifact.Status)
			}
		}
	}

	if len(deployResp.Warnings) > 0 {
		cmd.Println("\nWarnings:")
		for _, warning := range deployResp.Warnings {
			cmd.Printf("  ⚠ %s\n", warning)
		}
	}

	if deployResp.Status != "deployed" {
		return fmt.Errorf("deploy finished with status: %s", deployResp.Status)
	}

	cmd.Println("\n🎉 Deploy successful!")
	return nil
}
