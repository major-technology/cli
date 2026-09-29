package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/major-technology/cli/clients/workspace"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagRelatedKind string
	flagRelatedID   string
	flagRelatedJSON bool
)

var relatedArtifactsCmd = &cobra.Command{
	Use:   "related-artifacts",
	Short: "List the apps, agents, workflows, and skills connected to your artifact",
	Long: `Use this to see which other apps, agents, workflows, and skills point to the
artifact you're working on, and which ones it points to (up to two hops).

Defaults to the artifact in the current workspace. Pass --kind and --id to
look up any other artifact.`,
	Example: `  major related-artifacts
  major related-artifacts --json
  major related-artifacts --kind workflow --id 3f2c...`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRelatedArtifacts(cmd)
	},
}

func init() {
	relatedArtifactsCmd.Flags().StringVar(&flagRelatedKind, "kind", "", "Artifact kind: app, agent, workflow, or skill (defaults to the current workspace)")
	relatedArtifactsCmd.Flags().StringVar(&flagRelatedID, "id", "", "Artifact ID (defaults to the current workspace)")
	relatedArtifactsCmd.Flags().BoolVar(&flagRelatedJSON, "json", false, "Output in JSON format")
	relatedArtifactsCmd.MarkFlagsRequiredTogether("kind", "id")
}

func runRelatedArtifacts(cmd *cobra.Command) error {
	kind, id, err := resolveRelatedAnchor()
	if err != nil {
		return err
	}

	resp, err := singletons.GetAPIClient().GetRelatedArtifacts(kind, id)
	if err != nil {
		return clierrors.WrapError("failed to get related artifacts", err)
	}

	if flagRelatedJSON {
		return utils.WriteJSON(cmd, resp)
	}

	maxKind, maxName := len("KIND"), len("NAME")
	for _, a := range resp.Artifacts {
		maxKind = max(maxKind, len(a.Kind))
		maxName = max(maxName, len(a.Name))
	}
	cmd.Printf("%-*s  %-*s  %s\n", maxKind, "KIND", maxName, "NAME", "ID")
	for _, a := range resp.Artifacts {
		note := ""
		switch {
		case a.IsAnchor:
			note = "  (this artifact)"
		case !a.Accessible:
			note = "  (no access)"
		}
		cmd.Printf("%-*s  %-*s  %s%s\n", maxKind, a.Kind, maxName, a.Name, a.ID, note)
	}
	return nil
}

// resolveRelatedAnchor is --kind/--id when given, else the enclosing workspace's target.
func resolveRelatedAnchor() (string, string, error) {
	if flagRelatedKind != "" {
		return flagRelatedKind, flagRelatedID, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	_, cfg, err := workspace.Locate(cwd)
	if errors.Is(err, workspace.ErrNotFound) {
		return "", "", fmt.Errorf("no .major/config.json found; run this inside an app, agent, workflow, or skill workspace, or pass --kind and --id")
	}
	if err != nil {
		return "", "", err
	}
	return cfg.Target.Kind, cfg.Target.ID(), nil
}
