package agent

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "agent", Short: "Manage file-based agents"}

// bundle is the agent kind: agent.jsonc (or legacy agent.json) and prompt.md.
var bundle = target.BundleTarget{
	Kind:    "agent",
	APIPath: "agents",
	Files:   func(string) target.FileSet { return agentFiles },
	Pack:    target.PackZip,
}

var agentFiles = target.FileSet{
	Owns: func(rel string) bool {
		return rel == "agent.jsonc" || rel == "agent.json" || rel == "prompt.md"
	},
	Check: func(rels []string) error {
		if len(rels) != 2 || rels[1] != "prompt.md" || (rels[0] != "agent.json" && rels[0] != "agent.jsonc") {
			return fmt.Errorf("agent workspace requires prompt.md and exactly one of agent.jsonc or agent.json")
		}
		return nil
	},
}

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newListCmd(), newCreateCmd(), newCloneCmd())
	target.Register("agent", bundle)
}

type listResult struct{ *api.AgentListResponse }

func (r listResult) String() string {
	if len(r.Agents) == 0 {
		return "No agents found."
	}
	lines := make([]string, 0, len(r.Agents))
	for _, item := range r.Agents {
		status := "draft"
		if item.CurrentVersionID != nil {
			status = "published"
		}
		lines = append(lines, fmt.Sprintf("%s  %s  (%s)", item.ID, item.Name, status))
	}
	return strings.Join(lines, "\n")
}

func newListCmd() *cobra.Command {
	var readOnly bool
	cmd := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListAgents(org, readOnly)
		if err != nil {
			return err
		}
		return target.Output(cmd, listResult{result})
	}}
	cmd.Flags().BoolVar(&readOnly, "include-read-only", false, "Include agents you can use but not edit")
	return cmd
}
func newCreateCmd() *cobra.Command {
	var name, description string
	cmd := &cobra.Command{Use: "create <dir>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		dir, err := target.RequireNewDir(args[0])
		if err != nil {
			return err
		}
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		created, err := singletons.GetAPIClient().CreateAgent(org, name, description)
		if err != nil {
			return err
		}
		return bundle.Clone(cmd, org, workspace.Target{Kind: "agent", AgentID: created.AgentID}, dir)
	}}
	cmd.Flags().StringVar(&name, "name", "", "Agent name")
	cmd.Flags().StringVar(&description, "description", "", "Agent description")
	return cmd
}
func newCloneCmd() *cobra.Command {
	var id, dir string
	cmd := &cobra.Command{Use: "clone", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		if id == "" {
			if utils.IsNonInteractive(cmd) {
				return fmt.Errorf("major agent clone requires --id in non-interactive mode")
			}
			list, err := singletons.GetAPIClient().ListAgents(org, false)
			if err != nil {
				return err
			}
			if len(list.Agents) == 0 {
				return fmt.Errorf("no editable agents in this organization")
			}
			options := make([]huh.Option[string], 0, len(list.Agents))
			for _, item := range list.Agents {
				options = append(options, huh.NewOption(item.Name+" ("+item.ID+")", item.ID))
			}
			if err := huh.NewSelect[string]().Title("Choose an agent").Options(options...).Value(&id).Run(); err != nil {
				return err
			}
		}
		if dir == "" {
			dir = id
		}
		abs, err := target.RequireNewDir(dir)
		if err != nil {
			return err
		}
		return bundle.Clone(cmd, org, workspace.Target{Kind: "agent", AgentID: id}, abs)
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID")
	cmd.Flags().StringVar(&dir, "dir", "", "New directory (defaults to agent ID)")
	return cmd
}
