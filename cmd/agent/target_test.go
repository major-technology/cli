package agent

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/major-technology/cli/clients/workspace"
	"github.com/spf13/cobra"
)

type fakeTarget struct{ called []string }

func (f *fakeTarget) Pull(*TargetContext) (any, error) {
	f.called = append(f.called, "pull")
	return map[string]any{"action": "pull"}, nil
}
func (f *fakeTarget) Push(*TargetContext) (any, error) {
	f.called = append(f.called, "push")
	return map[string]any{"action": "push"}, nil
}
func (f *fakeTarget) Validate(*TargetContext) (any, error) {
	f.called = append(f.called, "validate")
	return map[string]any{"action": "validate"}, nil
}
func (f *fakeTarget) Publish(*TargetContext) (any, error) {
	f.called = append(f.called, "publish")
	return map[string]any{"action": "publish"}, nil
}

func TestPullOverwritesOnlyAgentFilesAndRejectsUnexpectedArchiveEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	writer := zip.NewWriter(&b)
	for name, content := range map[string]string{"agent.jsonc": "{}", "prompt.md": "hello"} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := unpackAgent(root, b.Bytes()); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "notes.txt")); err != nil || string(data) != "keep" {
		t.Fatalf("unrelated file: %s %v", data, err)
	}
	b.Reset()
	writer = zip.NewWriter(&b)
	file, err := writer.Create("../outside")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte("bad"))
	_ = writer.Close()
	if err := unpackAgent(root, b.Bytes()); err == nil {
		t.Fatal("traversal entry accepted")
	}
	if data, _ := os.ReadFile(filepath.Join(root, "prompt.md")); string(data) != "hello" {
		t.Fatalf("rejected archive changed workspace: %s", data)
	}
}

func TestRegisteredTargetRunsAllActionsWithoutRunnerChanges(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Write(root, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "skill", SkillID: "11111111-1111-4111-8111-111111111111"}}); err != nil {
		t.Fatal(err)
	}
	original, _ := os.Getwd()
	defer os.Chdir(original)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	f := &fakeTarget{}
	registry["skill"] = f
	defer delete(registry, "skill")
	for _, action := range []string{"pull", "push", "validate", "publish"} {
		cmd := &cobra.Command{Use: action}
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.Flags().Bool("json", true, "")
		if err := runTargetAction(cmd, action); err != nil {
			t.Fatalf("%s: %v", action, err)
		}
		if !bytes.Contains(out.Bytes(), []byte(`"action":"`+action+`"`)) {
			t.Fatalf("%s output: %q", action, out.String())
		}
	}
	if len(f.called) != 4 {
		t.Fatalf("calls=%v", f.called)
	}
}

func TestTextOutputIsReadableAndOmitsDownloadURL(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.Flags().Bool("json", false, "")
	if err := output(cmd, versionResult{AgentID: "a1", Name: "Helper", Version: 2, action: "Pulled"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Pulled Helper (version 2).\n" {
		t.Fatalf("text output: %q", out.String())
	}
	out.Reset()
	_ = cmd.Flags().Set("json", "true")
	if err := output(cmd, versionResult{AgentID: "a1", Name: "Helper", Version: 2, action: "Pulled"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "{\"agentId\":\"a1\",\"name\":\"Helper\",\"version\":2}\n" {
		t.Fatalf("json output: %q", out.String())
	}
}
