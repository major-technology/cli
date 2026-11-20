package function

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "function", Short: "Manage functions"}

const (
	entryFile  = "index.ts"
	configFile = "function.json"
	buildDir   = ".build"
)

// slugPattern is the server's rule for a function's slug, which is also its
// folder name.
var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// functionTarget is the function kind: a self-contained folder with index.ts,
// function.json, and its own package.json and clients. A push uploads the
// sources with the built bundle.
type functionTarget struct{ target.BundleTarget }

func (functionTarget) PublishPrompt() string {
	return "Publish the latest saved function version? Live runs of its workflow use it immediately."
}

var bundle = functionTarget{target.BundleTarget{
	Kind:             "function",
	APIPath:          "functions",
	Files:            func(string) target.FileSet { return functionFiles },
	ValidateByUpload: true,
	Build:            buildBundle,
}}

var skippedNames = map[string]bool{".DS_Store": true, "Thumbs.db": true}

func inBuildDir(rel string) bool { return strings.HasPrefix(rel, buildDir+"/") }

var functionFiles = target.FileSet{
	Owns: func(rel string) bool {
		if inBuildDir(rel) || skippedNames[path.Base(rel)] {
			return false
		}
		for _, segment := range strings.Split(path.Dir(rel), "/") {
			if target.SkippedDirs[segment] {
				return false
			}
		}
		return true
	},
	Check: func(rels []string) error {
		required := map[string]bool{entryFile: false, configFile: false}
		for _, rel := range rels {
			if _, ok := required[rel]; ok {
				required[rel] = true
			}
		}
		for _, name := range []string{entryFile, configFile} {
			if !required[name] {
				return fmt.Errorf("function workspace requires %s at the root", name)
			}
		}
		return nil
	},
	// A saved version keeps its built bundle beside the sources.
	Ignore: inBuildDir,
}

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newListCmd(), newCreateCmd(), newCloneCmd())
	target.Register("function", bundle)
}

// resolveWorkflow returns --workflow with the default organization, else the
// workflow workspace that encloses dir.
func resolveWorkflow(id, dir string) (workflowID, org string, err error) {
	if id != "" {
		org, err = target.OrganizationID()
		return id, org, err
	}
	_, cfg, err := workspace.Locate(dir)
	if err != nil && !errors.Is(err, workspace.ErrNotFound) {
		return "", "", err
	}
	if err != nil || cfg.Target.Kind != "workflow" {
		return "", "", fmt.Errorf("pass --workflow <workflowId>, or run this command inside a workflow folder")
	}
	return cfg.Target.WorkflowID, cfg.OrganizationID, nil
}

type listResult struct{ *api.FunctionListResponse }

func (r listResult) String() string {
	if len(r.Functions) == 0 {
		return "No functions found."
	}
	lines := make([]string, 0, len(r.Functions))
	for _, item := range r.Functions {
		status := "never saved"
		switch {
		case item.PublishedVersion != nil && item.Version != nil && *item.PublishedVersion == *item.Version:
			status = fmt.Sprintf("version %d, published", *item.Version)
		case item.PublishedVersion != nil && item.Version != nil:
			status = fmt.Sprintf("version %d, version %d published", *item.Version, *item.PublishedVersion)
		case item.Version != nil:
			status = fmt.Sprintf("version %d, unpublished", *item.Version)
		}
		lines = append(lines, fmt.Sprintf("%s  %s  (%s)", item.ID, item.Slug, status))
	}
	return strings.Join(lines, "\n")
}

func newListCmd() *cobra.Command {
	var workflowFlag string
	cmd := &cobra.Command{Use: "list", Short: "List a workflow's functions", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		workflowID, _, err := resolveWorkflow(workflowFlag, cwd)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListFunctions(workflowID)
		if err != nil {
			return err
		}
		return target.Output(cmd, listResult{result})
	}}
	cmd.Flags().StringVar(&workflowFlag, "workflow", "", "Workflow ID (defaults to the enclosing workflow folder)")
	return cmd
}

type createResult struct {
	FunctionID string `json:"functionId"`
	Slug       string `json:"slug"`
	Path       string `json:"path"`
}

func (r createResult) String() string {
	return fmt.Sprintf("Created function %s in %s.", r.Slug, r.Path)
}

// starterFiles is a new function: it builds and pushes as written.
func starterFiles(slug string) map[string]string {
	return map[string]string{
		entryFile:      "export default async function (input: Record<string, unknown>) {\n\treturn { input };\n}\n",
		configFile:     fmt.Sprintf("{\n\t\"label\": %q,\n\t\"description\": \"\"\n}\n", slug),
		"package.json": fmt.Sprintf("{\n\t\"name\": %q,\n\t\"private\": true,\n\t\"type\": \"module\",\n\t\"dependencies\": {}\n}\n", slug),
	}
}

// newCreateCmd names the function after its folder. A slug the workflow
// already has is cloned into the folder instead of started over.
func newCreateCmd() *cobra.Command {
	var workflowFlag string
	cmd := &cobra.Command{
		Use:   "create <dir>",
		Short: "Create a function in a workflow",
		Long: `Create a function and its folder. The folder name is the function's slug.

Inside a workflow folder, create it under functions/ so the workflow's
major push, validate, and publish include it:

  major function create functions/<slug>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := target.RequireNewDir(args[0])
			if err != nil {
				return err
			}
			slug := filepath.Base(dir)
			if !slugPattern.MatchString(slug) {
				return fmt.Errorf("function folder name %q must start with a lowercase letter and contain only a-z, 0-9 or - (at most 63 characters)", slug)
			}
			workflowID, org, err := resolveWorkflow(workflowFlag, filepath.Dir(dir))
			if err != nil {
				return err
			}
			created, err := singletons.GetAPIClient().CreateFunction(workflowID, slug)
			if err != nil {
				return err
			}
			wsTarget := workspace.Target{Kind: "function", FunctionID: created.FunctionID}
			if created.Version != nil {
				return bundle.Clone(cmd, org, wsTarget, dir)
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			for name, content := range starterFiles(slug) {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
					return err
				}
			}
			if err := workspace.Write(dir, workspace.Config{OrganizationID: org, Target: wsTarget}); err != nil {
				return err
			}
			return target.Output(cmd, createResult{FunctionID: created.FunctionID, Slug: created.Slug, Path: dir})
		},
	}
	cmd.Flags().StringVar(&workflowFlag, "workflow", "", "Workflow ID (defaults to the enclosing workflow folder)")
	return cmd
}

func newCloneCmd() *cobra.Command {
	var id, dir, workflowFlag string
	cmd := &cobra.Command{Use: "clone", Short: "Clone a function's latest saved version into a new folder", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		if id == "" {
			if utils.IsNonInteractive(cmd) {
				return fmt.Errorf("major function clone requires --id in non-interactive mode")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			workflowID, _, err := resolveWorkflow(workflowFlag, cwd)
			if err != nil {
				return err
			}
			list, err := singletons.GetAPIClient().ListFunctions(workflowID)
			if err != nil {
				return err
			}
			options := make([]huh.Option[string], 0, len(list.Functions))
			for _, item := range list.Functions {
				if item.Version != nil {
					options = append(options, huh.NewOption(item.Slug+" ("+item.ID+")", item.ID))
				}
			}
			if len(options) == 0 {
				return fmt.Errorf("this workflow has no saved functions")
			}
			if err := huh.NewSelect[string]().Title("Choose a function").Options(options...).Value(&id).Run(); err != nil {
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
		return bundle.Clone(cmd, org, workspace.Target{Kind: "function", FunctionID: id}, abs)
	}}
	cmd.Flags().StringVar(&id, "id", "", "Function ID")
	cmd.Flags().StringVar(&dir, "dir", "", "New directory (defaults to function ID)")
	cmd.Flags().StringVar(&workflowFlag, "workflow", "", "Workflow ID to choose a function from (defaults to the enclosing workflow folder)")
	return cmd
}
