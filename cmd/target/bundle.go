package target

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/transfer"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// PackFormat is how a bundle kind moves its files to and from the server.
type PackFormat int

const (
	// PackZip moves every owned file in one zip archive.
	PackZip PackFormat = iota
	// PackText moves one owned file as raw text.
	PackText
)

// FileSet decides which workspace files belong to a kind's bundle.
type FileSet struct {
	// Owns reports whether a slash-separated path under the root is bundle
	// content: push packs it, and pull may replace or remove it.
	Owns func(rel string) bool
	// Check is the bundle-level rule, run on the local files before an upload
	// and on an archive before it replaces anything.
	Check func(rels []string) error
	// Single is the one file of a PackText kind.
	Single string
}

// BundleTarget serves one bundle kind (agent, skill, workflow) over its
// /cli/<apiPath> routes.
type BundleTarget struct {
	Kind    string
	APIPath string
	// Files returns the kind's file set for one target id.
	Files func(id string) FileSet
	Pack  PackFormat
	// ValidateByUpload validates a zip by uploading it and sending its key,
	// instead of sending the files inline.
	ValidateByUpload bool
}

// SkippedDirs are never walked, packed, or replaced, on any kind.
var SkippedDirs = map[string]bool{".major": true, ".git": true, "node_modules": true, "__MACOSX": true}

const maxEntryBytes = 20 << 20

type versionResult struct {
	api.TargetVersion
	action string
}

func (r versionResult) String() string {
	return fmt.Sprintf("%s %s (version %d).", r.action, r.DisplayName(), r.Version)
}

type validateResult struct {
	*api.TargetValidateResponse
	kind string
}

func (r validateResult) String() string {
	lines := []string{capitalize(r.kind) + " bundle is valid."}
	for _, warning := range r.Warnings {
		lines = append(lines, "Warning: "+formatWarning(warning))
	}
	return strings.Join(lines, "\n")
}

type publishResult struct{ *api.TargetPublishResponse }

func (r publishResult) String() string {
	lines := []string{fmt.Sprintf("Published version %d.", r.Version)}
	for _, failure := range r.TriggerFailures {
		lines = append(lines, fmt.Sprintf("Trigger %s (%s: %s) was not registered: %s", failure.TriggerID, failure.ConnectorType, strings.Join(failure.Events, ", "), failure.Error))
	}
	return strings.Join(lines, "\n")
}

func (t BundleTarget) PublishPrompt() string {
	return fmt.Sprintf("Publish the latest saved %s version?", t.Kind)
}

func (t BundleTarget) Pull(ctx *TargetContext) (any, error) {
	id := ctx.Config.Target.ID()
	resp, err := ctx.API.PullTarget(t.APIPath, id)
	if err != nil {
		return nil, err
	}
	data, err := transfer.Download(resp.DownloadURL)
	if err != nil {
		return nil, err
	}
	files := t.Files(id)
	if t.Pack == PackText {
		name, err := singleFile(files)
		if err != nil {
			return nil, err
		}
		if err := writeFileUnder(ctx.Root, name, data); err != nil {
			return nil, err
		}
	} else if err := unpackBundle(ctx.Root, data, t.Kind, files); err != nil {
		return nil, err
	}
	resp.DownloadURL = ""
	return versionResult{TargetVersion: *resp, action: "Pulled"}, nil
}

func (t BundleTarget) Push(ctx *TargetContext) (any, error) {
	id := ctx.Config.Target.ID()
	data, contentType, err := t.pack(ctx.Root, id)
	if err != nil {
		return nil, err
	}
	key, err := t.upload(ctx, id, data, contentType)
	if err != nil {
		return nil, err
	}
	notes := ctx.Notes
	if t.Pack == PackText {
		notes = ""
	}
	pushed, err := ctx.API.PushTarget(t.APIPath, id, key, notes)
	if err != nil {
		return nil, err
	}
	pushed.DownloadURL = ""
	return versionResult{TargetVersion: *pushed, action: "Saved"}, nil
}

func (t BundleTarget) Validate(ctx *TargetContext) (any, error) {
	id := ctx.Config.Target.ID()
	var body map[string]any
	switch {
	case t.Pack == PackText:
		data, _, err := t.pack(ctx.Root, id)
		if err != nil {
			return nil, err
		}
		body = map[string]any{"definitionText": string(data)}
	case t.ValidateByUpload:
		data, contentType, err := t.pack(ctx.Root, id)
		if err != nil {
			return nil, err
		}
		key, err := t.upload(ctx, id, data, contentType)
		if err != nil {
			return nil, err
		}
		body = map[string]any{"uploadKey": key}
	default:
		rels, contents, err := readOwnedFiles(ctx.Root, t.Files(id))
		if err != nil {
			return nil, err
		}
		files := make([]api.AgentFile, 0, len(rels))
		for _, rel := range rels {
			files = append(files, api.AgentFile{Path: rel, Content: string(contents[rel])})
		}
		body = map[string]any{"files": files}
	}
	result, err := ctx.API.ValidateTarget(t.APIPath, id, body)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("%s bundle is invalid: %s", t.Kind, strings.Join(messages, "; "))
	}
	return validateResult{TargetValidateResponse: result, kind: t.Kind}, nil
}

func (t BundleTarget) Publish(ctx *TargetContext) (any, error) {
	if flag := ctx.Command.Flags().Lookup("slug"); flag != nil && flag.Changed {
		return nil, fmt.Errorf("--slug is only for app workspaces; %s workspaces have no URL", t.Kind)
	}
	published, err := ctx.API.PublishTarget(t.APIPath, ctx.Config.Target.ID())
	if err != nil {
		return nil, err
	}
	return publishResult{published}, nil
}

// pack reads the local bundle and returns the upload bytes and their content type.
func (t BundleTarget) pack(root, id string) ([]byte, string, error) {
	files := t.Files(id)
	if t.Pack == PackText {
		name, err := singleFile(files)
		if err != nil {
			return nil, "", err
		}
		data, err := readRegularFile(root, name)
		if err != nil {
			return nil, "", err
		}
		return data, "text/plain", nil
	}
	rels, contents, err := readOwnedFiles(root, files)
	if err != nil {
		return nil, "", err
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, rel := range rels {
		w, err := z.Create(rel)
		if err != nil {
			return nil, "", err
		}
		if _, err = w.Write(contents[rel]); err != nil {
			return nil, "", err
		}
	}
	if err := z.Close(); err != nil {
		return nil, "", err
	}
	return b.Bytes(), "application/zip", nil
}

func (t BundleTarget) upload(ctx *TargetContext, id string, data []byte, contentType string) (string, error) {
	upload, err := ctx.API.TargetUploadURL(t.APIPath, id)
	if err != nil {
		return "", err
	}
	if err := transfer.Upload(upload.UploadURL, contentType, data); err != nil {
		return "", err
	}
	return upload.UploadKey, nil
}

// singleFile is the one file of a PackText kind.
func singleFile(files FileSet) (string, error) {
	if files.Single == "" {
		return "", fmt.Errorf("text bundle kind has no single file")
	}
	return files.Single, nil
}

func readRegularFile(root, rel string) ([]byte, error) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%s not found; run major pull first", rel)
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file", rel)
	}
	return os.ReadFile(full)
}

// walkOwned lists the owned paths under root, including non-regular ones, so
// callers decide whether a symlink is an error (push) or just removed (pull).
func walkOwned(root string, files FileSet) (map[string]fs.FileMode, error) {
	found := map[string]fs.FileMode{}
	err := filepath.WalkDir(root, func(full string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if full == root {
			return nil
		}
		if d.IsDir() {
			if SkippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, full)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if files.Owns(rel) {
			found[rel] = d.Type()
		}
		return nil
	})
	return found, err
}

func readOwnedFiles(root string, files FileSet) ([]string, map[string][]byte, error) {
	owned, err := walkOwned(root, files)
	if err != nil {
		return nil, nil, err
	}
	rels := make([]string, 0, len(owned))
	for rel, mode := range owned {
		if !mode.IsRegular() {
			return nil, nil, fmt.Errorf("%s must be a regular file", rel)
		}
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	if err := files.Check(rels); err != nil {
		return nil, nil, err
	}
	contents := make(map[string][]byte, len(rels))
	for _, rel := range rels {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, nil, err
		}
		contents[rel] = data
	}
	return rels, contents, nil
}

// safeRel rejects archive paths that could leave the workspace root.
func safeRel(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || strings.ContainsRune(name, 0) {
		return false
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return path.Clean(name) == name
}

// unpackBundle replaces the owned files under root with the archive. Nothing
// changes until the whole archive has been read and checked.
func unpackBundle(root string, data []byte, kind string, files FileSet) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	incoming := map[string][]byte{}
	for _, entry := range reader.File {
		if strings.HasSuffix(entry.Name, "/") {
			continue
		}
		if !safeRel(entry.Name) || !files.Owns(entry.Name) {
			return fmt.Errorf("unexpected %s archive entry %q", kind, entry.Name)
		}
		if !entry.Mode().IsRegular() {
			return fmt.Errorf("%s archive entry %q is not a regular file", kind, entry.Name)
		}
		if _, exists := incoming[entry.Name]; exists {
			return fmt.Errorf("duplicate %s archive entry %q", kind, entry.Name)
		}
		if entry.UncompressedSize64 > maxEntryBytes {
			return fmt.Errorf("%s archive entry too large", kind)
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(io.LimitReader(in, maxEntryBytes+1))
		_ = in.Close()
		if err != nil {
			return err
		}
		if len(content) > maxEntryBytes {
			return fmt.Errorf("%s archive entry too large", kind)
		}
		incoming[entry.Name] = content
	}
	names := make([]string, 0, len(incoming))
	for name := range incoming {
		names = append(names, name)
	}
	sort.Strings(names)
	if err := files.Check(names); err != nil {
		return fmt.Errorf("%s archive: %w", kind, err)
	}
	owned, err := walkOwned(root, files)
	if err != nil {
		return err
	}
	for rel := range owned {
		if _, keep := incoming[rel]; keep {
			continue
		}
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, name := range names {
		if err := writeFileUnder(root, name, incoming[name]); err != nil {
			return err
		}
	}
	return nil
}

// writeFileUnder writes root/rel, refusing to follow a symlinked parent or file
// out of the workspace.
func writeFileUnder(root, rel string, content []byte) error {
	parts := strings.Split(rel, "/")
	dir := root
	for _, part := range parts[:len(parts)-1] {
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) {
			if err := os.Mkdir(dir, 0755); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("refusing to write %s: %s is not a real directory", rel, dir)
		}
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	if info, err := os.Lstat(full); err == nil && !info.Mode().IsRegular() {
		if err := os.Remove(full); err != nil {
			return err
		}
	}
	return os.WriteFile(full, content, 0644)
}

type cloneResult struct {
	AgentID    string `json:"agentId,omitempty"`
	SkillID    string `json:"skillId,omitempty"`
	WorkflowID string `json:"workflowId,omitempty"`
	Path       string `json:"path"`
	Version    int    `json:"version"`
	name       string
}

func (r cloneResult) String() string {
	return fmt.Sprintf("Cloned %s (version %d) into %s.", r.name, r.Version, r.Path)
}

// Clone writes .major/config.json into a new directory, pulls the latest saved
// version into it, and prints the clone result.
func (t BundleTarget) Clone(cmd *cobra.Command, org string, target workspace.Target, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cfg := workspace.Config{OrganizationID: org, Target: target}
	if err := workspace.Write(dir, cfg); err != nil {
		return err
	}
	result, err := t.Pull(&TargetContext{Root: dir, Config: &cfg, API: singletons.GetAPIClient(), Command: cmd})
	if err != nil {
		return err
	}
	pull := result.(versionResult)
	return Output(cmd, cloneResult{
		AgentID:    pull.AgentID,
		SkillID:    pull.SkillID,
		WorkflowID: pull.WorkflowID,
		Path:       dir,
		Version:    pull.Version,
		name:       pull.DisplayName(),
	})
}

// RequireNewDir returns the absolute path of dir, which must not exist yet.
func RequireNewDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(abs); err == nil {
		return "", fmt.Errorf("directory %s already exists", abs)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return abs, nil
}

func formatWarning(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var item api.TargetValidationError
	if err := json.Unmarshal(raw, &item); err == nil && item.Message != "" {
		if item.Path != "" {
			return item.Path + ": " + item.Message
		}
		return item.Message
	}
	return string(raw)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
