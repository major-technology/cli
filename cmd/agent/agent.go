package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "agent", Short: "Manage file-based agents"}

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newListCmd(), newCreateCmd(), newCloneCmd())
}

func organizationID() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if _, cfg, err := workspace.Locate(cwd); err == nil {
		return cfg.OrganizationID, nil
	} else if !errors.Is(err, workspace.ErrNotFound) {
		return "", err
	}
	id, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return "", fmt.Errorf("select a default organization with major org select")
	}
	return id, nil
}
func output(cmd *cobra.Command, result any) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}
	cmd.Println(result)
	return nil
}
func newListCmd() *cobra.Command {
	var readOnly bool
	cmd := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := organizationID()
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListAgents(org, readOnly)
		if err != nil {
			return err
		}
		return output(cmd, result)
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
		dir, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		if _, err := os.Lstat(dir); err == nil {
			return fmt.Errorf("directory %s already exists", dir)
		} else if !os.IsNotExist(err) {
			return err
		}
		org, err := organizationID()
		if err != nil {
			return err
		}
		created, err := singletons.GetAPIClient().CreateAgent(org, name, description)
		if err != nil {
			return err
		}
		return cloneToDirectory(cmd, org, created.AgentID, dir)
	}}
	cmd.Flags().StringVar(&name, "name", "", "Agent name")
	cmd.Flags().StringVar(&description, "description", "", "Agent description")
	return cmd
}
func newCloneCmd() *cobra.Command {
	var id, dir string
	cmd := &cobra.Command{Use: "clone", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := organizationID()
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
				options = append(options, huh.NewOption(item.Name+" ("+item.AgentID+")", item.AgentID))
			}
			if err := huh.NewSelect[string]().Title("Choose an agent").Options(options...).Value(&id).Run(); err != nil {
				return err
			}
		}
		if dir == "" {
			dir = id
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(abs); err == nil {
			return fmt.Errorf("directory %s already exists", abs)
		} else if !os.IsNotExist(err) {
			return err
		}
		return cloneToDirectory(cmd, org, id, abs)
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID")
	cmd.Flags().StringVar(&dir, "dir", "", "New directory (defaults to agent ID)")
	return cmd
}
func cloneToDirectory(cmd *cobra.Command, org, id, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cfg := workspace.Config{OrganizationID: org, Target: workspace.Target{Kind: "agent", AgentID: id}}
	if err := workspace.Write(dir, cfg); err != nil {
		return err
	}
	result, err := (&bundleTarget{}).Pull(&TargetContext{Root: dir, Config: &cfg, API: singletons.GetAPIClient(), Command: cmd})
	if err != nil {
		return err
	}
	pull := result.(*api.AgentPullResponse)
	return output(cmd, map[string]any{"agentId": pull.AgentID, "path": dir, "version": pull.Version})
}

func TargetCommands() []*cobra.Command {
	commands := make([]*cobra.Command, 0, 4)
	for _, action := range []string{"pull", "push", "validate", "publish"} {
		action := action
		cmd := &cobra.Command{Use: action, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			if action == "publish" {
				yes, _ := cmd.Flags().GetBool("yes")
				if !yes {
					if utils.IsNonInteractive(cmd) {
						return fmt.Errorf("major publish requires --yes in non-interactive mode")
					}
					var confirmed bool
					if err := huh.NewConfirm().Title("Publish the latest saved agent version?").Value(&confirmed).Run(); err != nil {
						return err
					}
					if !confirmed {
						return fmt.Errorf("publish cancelled")
					}
				}
			}
			return runTargetAction(cmd, action)
		}}
		cmd.Flags().Bool("json", false, "Print one JSON result")
		if action == "push" {
			cmd.Flags().StringP("message", "m", "", "Version notes")
		}
		if action == "publish" {
			cmd.Flags().Bool("yes", false, "Skip confirmation")
		}
		commands = append(commands, cmd)
	}
	return commands
}
