package agent

import (
	"fmt"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

type infoResult struct{ *api.AgentInfoResponse }

func (r infoResult) String() string {
	status := "not published"
	if r.PublishedVersionID != nil {
		status = "published"
	}
	lines := []string{fmt.Sprintf("%s  %s  (%s)", r.Name, r.ID, status)}
	if r.LatestVersion != nil {
		latest := fmt.Sprintf("Latest version: %d", r.LatestVersion.Version)
		if r.LatestVersion.Notes != nil && *r.LatestVersion.Notes != "" {
			latest += "  " + *r.LatestVersion.Notes
		}
		lines = append(lines, latest)
	}
	for _, key := range r.EnvKeys {
		state := "missing"
		if key.HasValue {
			state = "set"
		}
		lines = append(lines, fmt.Sprintf("%s  %s", key.Key, state))
	}
	return strings.Join(lines, "\n")
}

func newInfoCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: "info", Short: "Show an agent's published state, latest version, and env keys", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().GetAgentInfo(agentID)
		if err != nil {
			return err
		}
		return target.Output(cmd, infoResult{result})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	return cmd
}
