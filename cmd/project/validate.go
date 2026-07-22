package project

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/projects"
	"github.com/spf13/cobra"
)

// printIssues renders validation issues either as human-readable lines or JSON.
func printIssues(cmd *cobra.Command, issues []projects.Issue, asJSON bool) {
	if asJSON {
		out, _ := json.MarshalIndent(map[string]any{"valid": len(issues) == 0, "issues": issues}, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return
	}

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	for _, issue := range issues {
		location := issue.File
		if issue.Path != "" {
			location += " " + issue.Path
		}
		cmd.Println(errStyle.Render("✗ ") + location + ": " + issue.Message)
	}
}

func newValidateCmd() *cobra.Command {
	var dir string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the project definitions in a directory",
		Long:  `Validates project.json and every agent definition against the published schemas. Exits 0 when valid, 1 when not (CI-friendly).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			issues := projects.Validate(dir)

			if len(issues) > 0 {
				printIssues(cmd, issues, asJSON)
				return fmt.Errorf("%d validation issue(s)", len(issues))
			}

			if asJSON {
				printIssues(cmd, nil, true)
			} else {
				cmd.Println("✓ Project is valid")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Project directory to validate")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output machine-readable JSON")

	return cmd
}
