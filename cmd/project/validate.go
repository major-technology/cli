package project

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/projects"
	"github.com/spf13/cobra"
)

// printIssues renders validation issues either as human-readable lines or
// JSON. "valid" (JSON) and the pass/fail line (human) are computed from
// errors only - warnings are printed but never make a run invalid.
func printIssues(cmd *cobra.Command, issues []projects.Issue, asJSON bool) {
	if asJSON {
		errs, _ := projects.PartitionIssues(issues)
		out, _ := json.MarshalIndent(map[string]any{"valid": len(errs) == 0, "issues": issues}, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return
	}

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	for _, issue := range issues {
		location := issue.File
		if issue.Path != "" {
			location += " " + issue.Path
		}
		prefix := errStyle.Render("✗ ")
		if issue.IsWarning() {
			prefix = warnStyle.Render("⚠ ")
		}
		cmd.Println(prefix + location + ": " + issue.Message)
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
			errs, warnings := projects.PartitionIssues(issues)

			if len(issues) > 0 {
				printIssues(cmd, issues, asJSON)
			} else if asJSON {
				printIssues(cmd, nil, true)
			}

			if len(errs) > 0 {
				return fmt.Errorf("%d validation issue(s)", len(errs))
			}

			if !asJSON {
				if len(warnings) > 0 {
					cmd.Printf("✓ Project is valid (%d warning(s))\n", len(warnings))
				} else {
					cmd.Println("✓ Project is valid")
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Project directory to validate")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output machine-readable JSON")

	return cmd
}
