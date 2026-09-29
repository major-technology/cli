package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

type permissionsResult struct {
	permissions []api.AgentPermission
	app         bool
}

func (r permissionsResult) MarshalJSON() ([]byte, error) { return json.Marshal(r.permissions) }

func (r permissionsResult) String() string {
	if len(r.permissions) == 0 {
		return "No permissions found."
	}
	lines := make([]string, 0, len(r.permissions))
	for _, permission := range r.permissions {
		if r.app {
			name := permission.ToolName
			if permission.Matcher != nil {
				name = permission.Matcher.Method + " " + permission.Matcher.Path
			}
			lines = append(lines, fmt.Sprintf("%s  %s", name, permission.Decision))
			continue
		}
		line := fmt.Sprintf("%s  %s", permission.ToolName, permission.Decision)
		if permission.IsUpstream {
			line += "  (upstream)"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func newPermissionsCmd() *cobra.Command {
	var id, resourceID, applicationID string
	cmd := &cobra.Command{Use: "permissions", Short: "Show the agent's resolved tool or endpoint permissions on a resource or app", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if (resourceID == "") == (applicationID == "") {
			return fmt.Errorf("pass exactly one of --resource <resourceId> or --app <applicationId>")
		}
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		client := singletons.GetAPIClient()
		var permissions []api.AgentPermission
		if resourceID != "" {
			permissions, err = client.ListAgentResourcePermissions(agentID, resourceID)
		} else {
			permissions, err = client.ListAgentApplicationPermissions(agentID, applicationID)
		}
		if err != nil {
			return err
		}
		return target.Output(cmd, permissionsResult{permissions: permissions, app: applicationID != ""})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	cmd.Flags().StringVar(&resourceID, "resource", "", "Resource ID to show tool permissions for")
	cmd.Flags().StringVar(&applicationID, "app", "", "Application ID to show endpoint permissions for")
	return cmd
}
