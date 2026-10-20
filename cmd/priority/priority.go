package priority

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "priority", Short: "Manage Pepper priorities"}

// priorityFiles are the entries every priority folder must hold.
var priorityFiles = []string{"priority.json", "preview.html", "full.html"}

// bundle is the priority kind: every regular file under the root, less the skip list.
var bundle = target.BundleTarget{
	Kind:    "priority",
	APIPath: "priorities",
	Files: func(string) target.FileSet {
		return target.FileSet{
			Owns: func(rel string) bool {
				if path.Base(rel) == ".DS_Store" {
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
				present := map[string]bool{}
				for _, rel := range rels {
					present[rel] = true
				}
				for _, name := range priorityFiles {
					if !present[name] {
						return fmt.Errorf("%s: is required", name)
					}
				}
				return nil
			},
		}
	},
	Pack:             target.PackZip,
	ValidateByUpload: true,
	PublishMessage:   "priorities go live on push",
}

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newCreateCmd(), newCloneCmd(), newListCmd(), newDeleteCmd(), newRestoreCmd())
	target.Register("priority", bundle)
}

type createResult struct {
	PriorityID string `json:"priorityId"`
	Path       string `json:"path"`
}

func (r createResult) String() string {
	return fmt.Sprintf("Created priority %s in %s.", r.PriorityID, r.Path)
}

type priorityFile struct {
	Title    string `json:"title"`
	Position int    `json:"position"`
	Closed   bool   `json:"closed"`
}

// readTitle is the title in the folder's priority.json, or "" when there is none.
func readTitle(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "priority.json"))
	if err != nil {
		return ""
	}
	var file priorityFile
	if json.Unmarshal(data, &file) != nil {
		return ""
	}
	return file.Title
}

// scaffold writes the entries a folder is missing and leaves existing ones alone.
func scaffold(dir, title string) error {
	priority, err := json.MarshalIndent(priorityFile{Title: title}, "", "\t")
	if err != nil {
		return err
	}
	defaults := map[string][]byte{
		"priority.json": append(priority, '\n'),
		"preview.html":  []byte("<!doctype html>\n<html><body><h3>" + htmlEscape(title) + "</h3></body></html>\n"),
		"full.html":     []byte("<!doctype html>\n<html><body><h1>" + htmlEscape(title) + "</h1></body></html>\n"),
	}
	for _, name := range priorityFiles {
		full := filepath.Join(dir, name)
		if _, err := os.Lstat(full); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.WriteFile(full, defaults[name], 0644); err != nil {
			return err
		}
	}
	return nil
}

func htmlEscape(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}

func newCreateCmd() *cobra.Command {
	var dir, title string
	cmd := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if dir == "" && title == "" {
			return fmt.Errorf("major priority create requires --dir or --title")
		}
		if dir == "" {
			dir = slug(title)
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(abs, ".major", "config.json")); err == nil {
			return fmt.Errorf("directory %s is already a Major workspace", abs)
		}
		if title == "" {
			title = readTitle(abs)
		}
		if title == "" {
			title = filepath.Base(abs)
		}
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		created, err := singletons.GetAPIClient().CreatePriority(org, title)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(abs, 0755); err != nil {
			return err
		}
		if err := scaffold(abs, title); err != nil {
			return err
		}
		cfg := workspace.Config{OrganizationID: org, Target: workspace.Target{Kind: "priority", PriorityID: created.PriorityID}}
		if err := workspace.Write(abs, cfg); err != nil {
			return err
		}
		return target.Output(cmd, createResult{PriorityID: created.PriorityID, Path: abs})
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Folder to bind: an existing priority folder is adopted, missing entries are scaffolded")
	cmd.Flags().StringVar(&title, "title", "", "Priority title (defaults to the title in priority.json, then the folder name)")
	return cmd
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(text string) string {
	value := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if value == "" {
		return "priority"
	}
	return value
}

func newCloneCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{Use: "clone <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		if dir == "" {
			dir = args[0]
		}
		abs, err := target.RequireNewDir(dir)
		if err != nil {
			return err
		}
		return bundle.Clone(cmd, org, workspace.Target{Kind: "priority", PriorityID: args[0]}, abs)
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "New directory (defaults to the priority ID)")
	return cmd
}

type listResult struct{ *api.PriorityListResponse }

func (r listResult) String() string {
	if len(r.Open) == 0 && len(r.Closed) == 0 {
		return "No priorities found."
	}
	lines := make([]string, 0, len(r.Open)+len(r.Closed))
	for _, item := range r.Open {
		lines = append(lines, fmt.Sprintf("%s  %s  (open)", item.ID, item.Title))
	}
	for _, item := range r.Closed {
		lines = append(lines, fmt.Sprintf("%s  %s  (closed)", item.ID, item.Title))
	}
	return strings.Join(lines, "\n")
}

func newListCmd() *cobra.Command {
	return &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListPriorities(org)
		if err != nil {
			return err
		}
		return target.Output(cmd, listResult{result})
	}}
}

type deleteResult struct {
	PriorityID string `json:"priorityId"`
}

func (r deleteResult) String() string { return "Deleted priority " + r.PriorityID + "." }

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{Use: "delete <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := singletons.GetAPIClient().DeletePriority(args[0]); err != nil {
			return err
		}
		return target.Output(cmd, deleteResult{PriorityID: args[0]})
	}}
}

type restoredPriority struct {
	PriorityID string `json:"priorityId"`
	Path       string `json:"path"`
}
type restoreResult struct {
	Restored []restoredPriority `json:"restored"`
}

func (r restoreResult) String() string {
	return fmt.Sprintf("Restored %d priorities.", len(r.Restored))
}

// heldIDs are the priority ids bound by a folder directly under root.
func heldIDs(root string) (map[string]bool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	held := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name(), ".major", "config.json"))
		if err != nil {
			continue
		}
		var cfg workspace.Config
		if json.Unmarshal(data, &cfg) == nil && cfg.Target.PriorityID != "" {
			held[cfg.Target.PriorityID] = true
		}
	}
	return held, nil
}

func newRestoreCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{Use: "restore", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if root == "" {
			return fmt.Errorf("major priority restore requires --root")
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(abs, 0755); err != nil {
			return err
		}
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		list, err := singletons.GetAPIClient().ListPriorities(org)
		if err != nil {
			return err
		}
		held, err := heldIDs(abs)
		if err != nil {
			return err
		}
		result := restoreResult{Restored: []restoredPriority{}}
		for _, item := range list.Open {
			if held[item.ID] {
				continue
			}
			dir := filepath.Join(abs, slug(item.Title))
			if _, err := os.Lstat(dir); err == nil {
				dir += "-" + item.ID[:8]
			}
			cfg := workspace.Config{OrganizationID: org, Target: workspace.Target{Kind: "priority", PriorityID: item.ID}}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := workspace.Write(dir, cfg); err != nil {
				return err
			}
			if _, err := bundle.Pull(&target.TargetContext{Root: dir, Config: &cfg, API: singletons.GetAPIClient(), Command: cmd}); err != nil {
				return err
			}
			result.Restored = append(result.Restored, restoredPriority{PriorityID: item.ID, Path: dir})
		}
		return target.Output(cmd, result)
	}}
	cmd.Flags().StringVar(&root, "root", "", "Folder holding the priority folders")
	return cmd
}
