package workflow

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

var Cmd = &cobra.Command{Use: "workflow", Short: "Manage file-based workflows"}

// bundle is the workflow kind: one <workflowId>.jsonc definition, the same
// name the workflow pod uses.
var bundle = target.BundleTarget{
	Kind:    "workflow",
	APIPath: "workflows",
	Files:   workflowFiles,
	Pack:    target.PackText,
}

func workflowFiles(id string) target.FileSet {
	name := id + ".jsonc"
	return target.FileSet{
		Owns: func(rel string) bool { return rel == name },
		Check: func(rels []string) error {
			if len(rels) != 1 || rels[0] != name {
				return fmt.Errorf("workflow workspace requires %s", name)
			}
			return nil
		},
		Single: name,
	}
}

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newListCmd(), newCreateCmd(), newCloneCmd())
	target.Register("workflow", bundle)
}

type listResult struct{ *api.WorkflowListResponse }

func (r listResult) String() string {
	if len(r.Workflows) == 0 {
		return "No workflows found."
	}
	lines := make([]string, 0, len(r.Workflows))
	for _, item := range r.Workflows {
		status := "draft"
		if item.IsPublished {
			status = "published"
		}
		lines = append(lines, fmt.Sprintf("%s  %s  (%s)", item.WorkflowID, item.Label, status))
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
		result, err := singletons.GetAPIClient().ListWorkflows(org, readOnly)
		if err != nil {
			return err
		}
		return target.Output(cmd, listResult{result})
	}}
	cmd.Flags().BoolVar(&readOnly, "include-read-only", false, "Include workflows you can use but not edit")
	return cmd
}

// newCreateCmd takes no name: a workflow's label comes from its definition.
func newCreateCmd() *cobra.Command {
	return &cobra.Command{Use: "create <dir>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := target.RequireNewDir(args[0])
		if err != nil {
			return err
		}
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		created, err := singletons.GetAPIClient().CreateWorkflow(org)
		if err != nil {
			return err
		}
		return bundle.Clone(cmd, org, workspace.Target{Kind: "workflow", WorkflowID: created.WorkflowID}, dir)
	}}
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
				return fmt.Errorf("major workflow clone requires --id in non-interactive mode")
			}
			list, err := singletons.GetAPIClient().ListWorkflows(org, false)
			if err != nil {
				return err
			}
			if len(list.Workflows) == 0 {
				return fmt.Errorf("no editable workflows in this organization")
			}
			options := make([]huh.Option[string], 0, len(list.Workflows))
			for _, item := range list.Workflows {
				options = append(options, huh.NewOption(item.Label+" ("+item.WorkflowID+")", item.WorkflowID))
			}
			if err := huh.NewSelect[string]().Title("Choose a workflow").Options(options...).Value(&id).Run(); err != nil {
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
		return bundle.Clone(cmd, org, workspace.Target{Kind: "workflow", WorkflowID: id}, abs)
	}}
	cmd.Flags().StringVar(&id, "id", "", "Workflow ID")
	cmd.Flags().StringVar(&dir, "dir", "", "New directory (defaults to workflow ID)")
	return cmd
}
