package agent

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/transfer"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
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
type bundleTarget struct {
	kind    string
	apiPath string
	files   []string
}

var registry = map[string]Target{"agent": bundleTarget{kind: "agent", apiPath: "agents", files: []string{"agent.jsonc", "agent.json", "prompt.md"}}}

func runTargetAction(cmd *cobra.Command, action string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, cfg, err := workspace.Locate(cwd)
	if errors.Is(err, workspace.ErrNotFound) {
		return fmt.Errorf("no .major/config.json found; run major agent clone or major agent create")
	}
	if err != nil {
		return err
	}
	target, ok := registry[cfg.Target.Kind]
	if !ok {
		return fmt.Errorf("major %s supports agent targets only for now", action)
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
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}
	cmd.Println(result)
	return nil
}

func (t bundleTarget) Pull(ctx *TargetContext) (any, error) {
	resp, err := ctx.API.PullAgent(ctx.Config.Target.ID())
	if err != nil {
		return nil, err
	}
	data, err := transfer.Download(resp.DownloadURL)
	if err != nil {
		return nil, err
	}
	if err := unpackAgent(ctx.Root, data); err != nil {
		return nil, err
	}
	return resp, nil
}
func (t bundleTarget) Push(ctx *TargetContext) (any, error) {
	files, err := readAgentFiles(ctx.Root)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, file := range files {
		w, err := z.Create(file.Path)
		if err != nil {
			return nil, err
		}
		if _, err = w.Write([]byte(file.Content)); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	upload, err := ctx.API.AgentUploadURL(ctx.Config.Target.ID())
	if err != nil {
		return nil, err
	}
	if err := transfer.Upload(upload.UploadURL, b.Bytes()); err != nil {
		return nil, err
	}
	return ctx.API.PushAgent(ctx.Config.Target.ID(), upload.UploadKey, ctx.Notes)
}
func (t bundleTarget) Validate(ctx *TargetContext) (any, error) {
	files, err := readAgentFiles(ctx.Root)
	if err != nil {
		return nil, err
	}
	result, err := ctx.API.ValidateAgent(ctx.Config.Target.ID(), files)
	if err != nil {
		return nil, err
	}
	if !result.Valid {
		messages := make([]string, 0, len(result.Errors))
		for _, item := range result.Errors {
			messages = append(messages, item.Message)
		}
		return nil, fmt.Errorf("agent bundle is invalid: %s", strings.Join(messages, "; "))
	}
	return result, nil
}
func (t bundleTarget) Publish(ctx *TargetContext) (any, error) {
	return ctx.API.PublishAgent(ctx.Config.Target.ID())
}

func readAgentFiles(root string) ([]api.AgentFile, error) {
	var files []api.AgentFile
	for _, name := range []string{"agent.jsonc", "agent.json", "prompt.md"} {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s must be a regular file", name)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, api.AgentFile{Path: name, Content: string(data)})
	}
	if len(files) != 2 {
		return nil, fmt.Errorf("agent workspace requires prompt.md and exactly one of agent.jsonc or agent.json")
	}
	if files[0].Path == "prompt.md" || files[1].Path != "prompt.md" {
		return nil, fmt.Errorf("agent workspace requires prompt.md and exactly one agent definition")
	}
	return files, nil
}
func unpackAgent(root string, data []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	for _, entry := range reader.File {
		if entry.Name != "agent.jsonc" && entry.Name != "agent.json" && entry.Name != "prompt.md" {
			return fmt.Errorf("unexpected agent archive entry %q", entry.Name)
		}
		if _, exists := files[entry.Name]; exists {
			return fmt.Errorf("duplicate agent archive entry %q", entry.Name)
		}
		if entry.UncompressedSize64 > 20<<20 {
			return fmt.Errorf("agent archive entry too large")
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(io.LimitReader(in, (20<<20)+1))
		_ = in.Close()
		if err != nil {
			return err
		}
		if len(content) > 20<<20 {
			return fmt.Errorf("agent archive entry too large")
		}
		files[entry.Name] = content
	}
	if len(files) != 2 || files["prompt.md"] == nil || (files["agent.jsonc"] == nil && files["agent.json"] == nil) {
		return fmt.Errorf("agent archive requires prompt.md and exactly one definition")
	}
	for _, name := range []string{"agent.jsonc", "agent.json", "prompt.md"} {
		if err := os.Remove(filepath.Join(root, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), content, 0644); err != nil {
			return err
		}
	}
	return nil
}
