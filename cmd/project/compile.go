package project

import (
	"encoding/json"
	"fmt"

	"github.com/major-technology/cli/projects"
	"github.com/spf13/cobra"
)

func newCompileCmd() *cobra.Command {
	var dir string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile the project into its canonical config JSON",
		Long:  `Validates and compiles the project directory, printing the compiled config JSON to stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, issues := projects.Compile(dir)

			if len(issues) > 0 {
				printIssues(cmd, issues, asJSON)
				return fmt.Errorf("%d validation issue(s)", len(issues))
			}

			if asJSON {
				cmd.Println(string(result.ConfigJSON))
				return nil
			}

			pretty, err := json.MarshalIndent(result.Config, "", "  ")
			if err != nil {
				return err
			}
			cmd.Println(string(pretty))
			cmd.Printf("\nconfig hash: %s\n", result.Hash)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Project directory to compile")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print the canonical single-line JSON only (machine-readable)")

	return cmd
}
