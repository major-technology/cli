package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// infoResult adds the workspace's local version, when this command runs
// inside the folder of the same agent, without adding it to --json output.
type infoResult struct {
	*api.AgentInfoResponse
	localVersion int
	hasLocal     bool
}

func (r infoResult) String() string {
	status := "not published"
	if r.CurrentVersionID != nil {
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
	if r.hasLocal {
		if r.LatestVersion != nil && r.localVersion < r.LatestVersion.Version {
			lines = append(lines, fmt.Sprintf("Local version: %d (behind latest %d; run major pull)", r.localVersion, r.LatestVersion.Version))
		} else {
			lines = append(lines, fmt.Sprintf("Local version: %d (up to date)", r.localVersion))
		}
	}
	return strings.Join(lines, "\n")
}

func newInfoCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: "info", Short: "Show an agent's published state and latest version", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().GetAgentInfo(agentID)
		if err != nil {
			return err
		}
		info := infoResult{AgentInfoResponse: result}
		if version, ok := localAgentVersion(agentID); ok {
			info.hasLocal = true
			info.localVersion = version
		}
		return target.Output(cmd, info)
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	return cmd
}

// localAgentVersion is the version this workspace last pulled or pushed, when
// the current directory is the folder of this same agent. A folder written
// before versions were recorded has none (0), so it reports nothing.
func localAgentVersion(agentID string) (int, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return 0, false
	}
	_, cfg, err := workspace.Locate(cwd)
	if err != nil {
		return 0, false
	}
	if cfg.Target.Kind != "agent" || cfg.Target.AgentID != agentID || cfg.Target.Version == 0 {
		return 0, false
	}
	return cfg.Target.Version, true
}
