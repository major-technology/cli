package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/transfer"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "workflow", Short: "Manage file-based workflows"}

// workflowTarget is the workflow kind: a <workflowId>.jsonc definition, the
// same name the workflow pod uses, moved as text, and every other workspace
// file moved beside it as one archive. Validate and publish are the shared
// text-bundle ones.
type workflowTarget struct{ target.BundleTarget }

var bundle = workflowTarget{target.BundleTarget{
	Kind:    "workflow",
	APIPath: "workflows",
	Files:   definitionFiles,
	Pack:    target.PackText,
}}

// NestedDirs makes each folder under functions/ a workspace of its own:
// major push, validate, and publish run on all of them before the workflow.
func (workflowTarget) NestedDirs() map[string]string {
	return map[string]string{"functions": "function"}
}

func definitionName(id string) string { return id + ".jsonc" }

func definitionFiles(id string) target.FileSet {
	name := definitionName(id)
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

var skippedNames = map[string]bool{".DS_Store": true, "Thumbs.db": true}

// archiveFiles is everything in the workspace but the definition: the nested
// workspaces' sources and their .major/config.json, without build output.
func archiveFiles(id string) target.FileSet {
	name := definitionName(id)
	return target.FileSet{
		Owns: func(rel string) bool {
			if rel == name || skippedNames[path.Base(rel)] {
				return false
			}
			segments := strings.Split(path.Dir(rel), "/")
			for i, segment := range segments {
				if segment == ".build" {
					return false
				}
				if segment == workspace.ConfigDirName {
					// Only a nested workspace's config file, never the root's.
					return i > 0 && i == len(segments)-1 && path.Base(rel) == "config.json"
				}
				if target.SkippedDirs[segment] {
					return false
				}
			}
			return true
		},
		Check:         func([]string) error { return nil },
		NestedConfigs: true,
	}
}

// Pull writes the definition and replaces the rest of the workspace with the
// version's archive. A version saved without an archive leaves other files alone.
func (t workflowTarget) Pull(ctx *target.TargetContext) (any, error) {
	id := ctx.Config.Target.ID()
	resp, err := ctx.API.PullWorkflow(id)
	if err != nil {
		return nil, err
	}
	definition, err := transfer.Download(resp.DownloadURL)
	if err != nil {
		return nil, err
	}
	if resp.ArchiveDownloadURL != nil {
		archive, err := transfer.Download(*resp.ArchiveDownloadURL)
		if err != nil {
			return nil, err
		}
		if err := target.Unpack(ctx.Root, archive, t.Kind, archiveFiles(id)); err != nil {
			return nil, err
		}
	}
	if err := target.WriteFile(ctx.Root, definitionName(id), definition); err != nil {
		return nil, err
	}
	ctx.Config.Target.PushedHash = ""
	if err := target.RecordVersion(ctx, resp.Version); err != nil {
		return nil, err
	}
	resp.DownloadURL = ""
	return target.VersionResult(resp.TargetVersion, "Pulled"), nil
}

// Push saves the definition and the archive as one new version. Nested
// workspaces have already been pushed by the time it runs.
func (t workflowTarget) Push(ctx *target.TargetContext) (any, error) {
	id := ctx.Config.Target.ID()
	definition, err := target.ReadFile(ctx.Root, definitionName(id))
	if err != nil {
		return nil, err
	}
	archive, err := target.PackFiles(ctx.Root, archiveFiles(id))
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	hasher.Write(definition)
	hasher.Write([]byte{0})
	hasher.Write(archive)
	hash := hex.EncodeToString(hasher.Sum(nil))
	if ctx.Config.Target.Version > 0 && ctx.Config.Target.PushedHash == hash {
		return target.VersionResult(api.TargetVersion{Version: ctx.Config.Target.Version}, "Unchanged"), nil
	}
	upload, err := ctx.API.WorkflowUploadURL(id)
	if err != nil {
		return nil, err
	}
	if err := transfer.Upload(upload.UploadURL, "text/plain", definition); err != nil {
		return nil, err
	}
	// An API that stores no archive offers no URL for one.
	if upload.ArchiveURL != "" {
		if err := transfer.Upload(upload.ArchiveURL, "application/zip", archive); err != nil {
			return nil, err
		}
	}
	pushed, err := ctx.API.PushWorkflow(id, upload.UploadKey, upload.ArchiveKey)
	if err != nil {
		return nil, err
	}
	ctx.Config.Target.PushedHash = hash
	if err := target.RecordVersion(ctx, pushed.Version); err != nil {
		return nil, err
	}
	return target.VersionResult(*pushed, "Saved"), nil
}

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newListCmd(), newCreateCmd(), newCloneCmd(), newRunCmd(), newEventsCmd())
	target.Register("workflow", bundle)
}

// resolveWorkflowID returns --id, else the workflow of the .major/config.json workspace this
// command runs in.
func resolveWorkflowID(id string) (string, error) {
	if id != "" {
		return id, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	_, cfg, err := workspace.Locate(cwd)
	if err != nil && !errors.Is(err, workspace.ErrNotFound) {
		return "", err
	}
	if err != nil || cfg.Target.Kind != "workflow" {
		return "", fmt.Errorf("pass --id <workflowId>, or run this command inside a workflow folder (one with .major/config.json for a workflow)")
	}
	return cfg.Target.WorkflowID, nil
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
		lines = append(lines, fmt.Sprintf("%s  %s  (%s)", item.ID, item.Label, status))
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
				options = append(options, huh.NewOption(item.Label+" ("+item.ID+")", item.ID))
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
