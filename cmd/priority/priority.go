package priority

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/file"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// Cmd holds the commands for what is Pepper's about a priority: its row, title, position and closed
// state. A priority's content is a File workspace, moved with `major push` and `major pull`.
var Cmd = &cobra.Command{Use: "priority", Short: "Manage Pepper priorities"}

const priorityFileName = "priority.json"

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newCreateCmd(), newValidateCmd(), newUpdateCmd(), newListCmd(), newDeleteCmd(), newRestoreCmd())
}

// withJSONFailure prints a failed command as one JSON line on stderr under --json, like the target commands.
func withJSONFailure(run func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := run(cmd, args); err != nil {
			return target.JSONFailure(cmd, err)
		}
		return nil
	}
}

type priorityFile struct {
	Title    string `json:"title"`
	Position int    `json:"position"`
	Closed   bool   `json:"closed"`
}

type createResult struct {
	FileID string `json:"fileId"`
	Path   string `json:"path"`
}

func (r createResult) String() string {
	return fmt.Sprintf("Created priority %s in %s.", r.FileID, r.Path)
}

// readTitle is the title in the folder's priority.json, or "" when there is none.
func readTitle(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, priorityFileName))
	if err != nil {
		return ""
	}
	var parsed priorityFile
	if json.Unmarshal(data, &parsed) != nil {
		return ""
	}
	return parsed.Title
}

func htmlEscape(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}

// scaffold writes the entries a folder is missing and leaves existing ones alone.
func scaffold(dir, title string) error {
	priority, err := json.MarshalIndent(priorityFile{Title: title}, "", "\t")
	if err != nil {
		return err
	}
	defaults := []struct {
		name    string
		content []byte
	}{
		{priorityFileName, append(priority, '\n')},
		{"preview.html", []byte(previewScaffold(title))},
		{"full.html", []byte("<!doctype html>\n<html><body><h1>" + htmlEscape(title) + "</h1></body></html>\n")},
	}
	for _, entry := range defaults {
		full := filepath.Join(dir, entry.name)
		if _, err := os.Lstat(full); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.WriteFile(full, entry.content, 0644); err != nil {
			return err
		}
	}
	return nil
}

// previewScaffold is a placeholder preview: the title and one status line, nothing more, so it
// suggests no layout. It draws no border or padding (the card does) and uses the injected theme variables.
func previewScaffold(title string) string {
	return `<!doctype html>
<html>
<body style="display:flex;flex-direction:column;gap:4px;font-family:var(--font)">
<span style="font-size:14px;font-weight:500;color:var(--fg)">` + htmlEscape(title) + `</span>
<span style="font-size:12px;color:var(--muted)">Just created</span>
</body>
</html>
`
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(text string) string {
	value := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if value == "" {
		return "priority"
	}
	return value
}

func newCreateCmd() *cobra.Command {
	var dir, title string
	cmd := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: withJSONFailure(func(cmd *cobra.Command, _ []string) error {
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
		cfg := workspace.Config{OrganizationID: org, Target: workspace.Target{Kind: "file", FileID: created.FileID}}
		if err := workspace.Write(abs, cfg); err != nil {
			return err
		}
		return target.Output(cmd, createResult{FileID: created.FileID, Path: abs})
	})}
	cmd.Flags().StringVar(&dir, "dir", "", "Folder to bind: an existing priority folder is adopted, missing entries are scaffolded")
	cmd.Flags().StringVar(&title, "title", "", "Priority title (defaults to the title in priority.json, then the folder name)")
	return cmd
}

// folderFileID is the File id a priority folder is bound to.
func folderFileID(dir string) (string, string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	cfg, err := workspace.Load(abs)
	if err != nil {
		return "", "", fmt.Errorf("%s is not a priority folder: %w", abs, err)
	}
	if cfg.Target.Kind != "file" {
		return "", "", fmt.Errorf("%s is not a priority folder: its workspace is a %s", abs, cfg.Target.Kind)
	}
	return cfg.Target.FileID, abs, nil
}

type validateResult struct{}

func (validateResult) String() string { return "Priority folder is valid." }

func newValidateCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{Use: "validate", Args: cobra.NoArgs, RunE: withJSONFailure(func(cmd *cobra.Command, _ []string) error {
		id, abs, err := folderFileID(dir)
		if err != nil {
			return err
		}
		entries, err := target.ListOwned(abs, file.Bundle.Files(id))
		if err != nil {
			return err
		}
		var priorityJSON *string
		if data, err := os.ReadFile(filepath.Join(abs, priorityFileName)); err == nil {
			text := string(data)
			priorityJSON = &text
		} else if !os.IsNotExist(err) {
			return err
		}
		result, err := singletons.GetAPIClient().ValidatePriority(id, priorityJSON, entries)
		if err != nil {
			return err
		}
		if !result.Valid {
			messages := make([]string, 0, len(result.Errors))
			for _, item := range result.Errors {
				if item.Path != "" {
					messages = append(messages, item.Path+": "+item.Message)
				} else {
					messages = append(messages, item.Message)
				}
			}
			return &target.InvalidBundleError{Err: fmt.Errorf("%s", strings.Join(messages, "; "))}
		}
		return target.Output(cmd, validateResult{})
	})}
	cmd.Flags().StringVar(&dir, "dir", ".", "Priority folder")
	return cmd
}

type updateResult struct {
	FileID string `json:"fileId"`
}

func (r updateResult) String() string { return "Updated priority " + r.FileID + "." }

func newUpdateCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{Use: "update <fileId>", Args: cobra.ExactArgs(1), RunE: withJSONFailure(func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(filepath.Join(dir, priorityFileName))
		if err != nil {
			return err
		}
		var priority map[string]any
		if err := json.Unmarshal(data, &priority); err != nil {
			return &target.InvalidBundleError{Err: fmt.Errorf("%s: is not valid JSON", priorityFileName)}
		}
		if err := singletons.GetAPIClient().UpdatePriority(args[0], priority); err != nil {
			return err
		}
		return target.Output(cmd, updateResult{FileID: args[0]})
	})}
	cmd.Flags().StringVar(&dir, "dir", ".", "Priority folder holding priority.json")
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
	return &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: withJSONFailure(func(cmd *cobra.Command, _ []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListPriorities(org)
		if err != nil {
			return err
		}
		return target.Output(cmd, listResult{result})
	})}
}

type deleteResult struct {
	FileID string `json:"fileId"`
}

func (r deleteResult) String() string { return "Deleted priority " + r.FileID + "." }

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{Use: "delete <fileId>", Args: cobra.ExactArgs(1), RunE: withJSONFailure(func(cmd *cobra.Command, args []string) error {
		if err := singletons.GetAPIClient().DeletePriority(args[0]); err != nil {
			return err
		}
		return target.Output(cmd, deleteResult{FileID: args[0]})
	})}
}

type restoredPriority struct {
	FileID string `json:"fileId"`
	Path   string `json:"path"`
}
type restoreResult struct {
	Restored []restoredPriority `json:"restored"`
}

func (r restoreResult) String() string {
	return fmt.Sprintf("Restored %d priorities.", len(r.Restored))
}

// heldIDs are the File ids bound by a folder directly under root.
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
		if json.Unmarshal(data, &cfg) == nil && cfg.Target.FileID != "" {
			held[cfg.Target.FileID] = true
		}
	}
	return held, nil
}

func newRestoreCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{Use: "restore", Args: cobra.NoArgs, RunE: withJSONFailure(func(cmd *cobra.Command, _ []string) error {
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
			cfg := workspace.Config{OrganizationID: org, Target: workspace.Target{Kind: "file", FileID: item.ID}}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := workspace.Write(dir, cfg); err != nil {
				return err
			}
			if _, err := file.Bundle.Pull(&target.TargetContext{Root: dir, Config: &cfg, API: singletons.GetAPIClient(), Command: cmd}); err != nil {
				return err
			}
			// priority.json is not in the File bundle: it comes back from the priority's row.
			data, err := json.MarshalIndent(priorityFile{Title: item.Title, Position: item.Position, Closed: false}, "", "\t")
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, priorityFileName), append(data, '\n'), 0644); err != nil {
				return err
			}
			result.Restored = append(result.Restored, restoredPriority{FileID: item.ID, Path: dir})
		}
		return target.Output(cmd, result)
	})}
	cmd.Flags().StringVar(&root, "root", "", "Folder holding the priority folders")
	return cmd
}
