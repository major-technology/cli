package target

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type TargetContext struct {
	Root    string
	Config  *workspace.Config
	API     *api.Client
	Command *cobra.Command
	Notes   string
}
type Target interface {
	Pull(*TargetContext) (any, error)
	Push(*TargetContext) (any, error)
	Validate(*TargetContext) (any, error)
	Publish(*TargetContext) (any, error)
}

// PublishPrompter is an optional Target method: the confirmation `major publish`
// asks before it runs without --yes.
type PublishPrompter interface {
	PublishPrompt() string
}

// FlagAdder is an optional Target method for kind-specific flags on a
// top-level command, e.g. the app-only `major publish --slug`.
type FlagAdder interface {
	AddFlags(action string, flags *pflag.FlagSet)
}

var registry = map[string]Target{}

// Register makes `major pull|push|validate|publish` serve workspaces whose
// .major/config.json names this kind.
func Register(kind string, t Target) {
	registry[kind] = t
}

var errNoWorkspace = errors.New("no .major/config.json found; run major agent clone, major skill clone, major workflow clone, or major app clone (or major agent create, major skill create, major workflow create, or major app create)")

func runTargetAction(cmd *cobra.Command, action string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, cfg, err := workspace.Locate(cwd)
	if errors.Is(err, workspace.ErrNotFound) {
		return errNoWorkspace
	}
	if err != nil {
		return err
	}
	target, ok := registry[cfg.Target.Kind]
	if !ok {
		return fmt.Errorf("major %s does not support %s targets", action, cfg.Target.Kind)
	}
	ctx := &TargetContext{Root: root, Config: cfg, API: singletons.GetAPIClient(), Command: cmd}
	if flag := cmd.Flags().Lookup("message"); flag != nil {
		ctx.Notes, _ = cmd.Flags().GetString("message")
	}
	var result any
	switch action {
	case "pull":
		result, err = target.Pull(ctx)
	case "push":
		result, err = target.Push(ctx)
	case "validate":
		result, err = target.Validate(ctx)
	case "publish":
		result, err = target.Publish(ctx)
	default:
		return fmt.Errorf("unknown target action %q", action)
	}
	if err != nil {
		return err
	}
	return Output(cmd, result)
}

// Output writes one JSON object with --json, and the result's text otherwise.
func Output(cmd *cobra.Command, result any) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		return utils.WriteJSON(cmd, result)
	}
	cmd.Println(result)
	return nil
}

// OrganizationID is the org of the enclosing workspace, or the default org.
func OrganizationID() (string, error) {
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

// publishPrompt names the kind of the enclosing workspace. A missing or broken
// workspace gets a generic prompt; runTargetAction then reports the error.
func publishPrompt() string {
	const generic = "Publish the latest saved version?"
	cwd, err := os.Getwd()
	if err != nil {
		return generic
	}
	_, cfg, err := workspace.Locate(cwd)
	if err != nil {
		return generic
	}
	if prompter, ok := registry[cfg.Target.Kind].(PublishPrompter); ok {
		return prompter.PublishPrompt()
	}
	return generic
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
					if err := huh.NewConfirm().Title(publishPrompt()).Value(&confirmed).Run(); err != nil {
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
			cmd.Long = `Save the workspace as a new version of its target.

For a skill, workflow, or agent, push uploads the workspace as a new unpublished
version. -m sets the version notes.

For an app, push sends local commits to the default branch:
  - With uncommitted changes and -m, push stages every change in the repository
    (git add -A), commits it with -m as the message, then pushes.
  - With uncommitted changes and no -m, push fails and commits nothing.
  - With a clean tree, push pushes HEAD and commits nothing.

To push only some of your changes, stage and commit them with git, then run
major push with a clean tree.`
			cmd.Flags().StringP("message", "m", "", "Version notes, or for an app, the commit message for ALL uncommitted changes (git add -A)")
		}
		if action == "publish" {
			cmd.Flags().Bool("yes", false, "Skip confirmation")
		}
		kinds := make([]string, 0, len(registry))
		for kind := range registry {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		for _, kind := range kinds {
			if adder, ok := registry[kind].(FlagAdder); ok {
				adder.AddFlags(action, cmd.Flags())
			}
		}
		commands = append(commands, cmd)
	}
	return commands
}
