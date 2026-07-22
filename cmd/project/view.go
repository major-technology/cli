package project

import (
	"encoding/json"

	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

func newViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Show the project's latest compiled state",
		Long:  `Shows the project, its latest compiled version, and the compiled config the platform holds for it.`,
		PreRunE: middleware.Compose(
			middleware.CheckLogin,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runView(cmd)
		},
	}

	return cmd
}

func runView(cmd *cobra.Command) error {
	projectID, _, err := getProjectAndOrgID()
	if err != nil {
		return err
	}

	apiClient := singletons.GetAPIClient()

	resp, err := apiClient.GetProject(projectID)
	if err != nil {
		return err
	}

	cmd.Printf("Project:    %s\n", resp.Name)
	cmd.Printf("Repository: %s\n", resp.RepositoryName)

	if resp.LatestVersion == nil {
		cmd.Println("\nNo compiled versions yet. Push to main to trigger a compile.")
		return nil
	}

	cmd.Printf("\nLatest version: %s (%s)\n", shortHash(resp.LatestVersion.CommitHash), resp.LatestVersion.CompileStatus)

	if resp.LatestVersion.CompileStatus == "failed" {
		cmd.Printf("Compile error:\n%s\n", resp.LatestVersion.CompileError)
		return nil
	}

	if len(resp.CompiledConfig) > 0 {
		var pretty json.RawMessage = resp.CompiledConfig
		out, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			cmd.Printf("\n%s\n", string(out))
		}
	}

	return nil
}
