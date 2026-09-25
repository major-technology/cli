package target

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
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

var testFiles = FileSet{
	Owns: func(rel string) bool { return rel == "agent.jsonc" || rel == "agent.json" || rel == "prompt.md" },
	Check: func(rels []string) error {
		if len(rels) != 2 || rels[1] != "prompt.md" {
			return fmt.Errorf("needs a definition and prompt.md")
		}
		return nil
	},
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	writer := zip.NewWriter(&b)
	for name, content := range files {
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
	return b.Bytes()
}

func TestPullOverwritesOnlyOwnedFilesAndRejectsUnexpectedArchiveEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := unpackBundle(root, zipOf(t, map[string]string{"agent.jsonc": "{}", "prompt.md": "hello"}), "agent", testFiles); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "notes.txt")); err != nil || string(data) != "keep" {
		t.Fatalf("unrelated file: %s %v", data, err)
	}
	if err := unpackBundle(root, zipOf(t, map[string]string{"../outside": "bad"}), "agent", testFiles); err == nil {
		t.Fatal("traversal entry accepted")
	}
	if data, _ := os.ReadFile(filepath.Join(root, "prompt.md")); string(data) != "hello" {
		t.Fatalf("rejected archive changed workspace: %s", data)
	}
}

func TestPullRemovesOwnedFilesMissingFromTheNewBundle(t *testing.T) {
	root := t.TempDir()
	owned := FileSet{
		Owns:  func(rel string) bool { return !strings.HasPrefix(rel, ".major/") },
		Check: func([]string) error { return nil },
	}
	for rel, content := range map[string]string{"old.md": "stale", "SKILL.md": "v1", ".major/config.json": "{}"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := unpackBundle(root, zipOf(t, map[string]string{"SKILL.md": "v2", "refs/a.md": "a"}), "skill", owned); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "old.md")); !os.IsNotExist(err) {
		t.Fatalf("stale owned file kept: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(root, "refs/a.md")); string(data) != "a" {
		t.Fatalf("nested file: %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(root, ".major/config.json")); string(data) != "{}" {
		t.Fatalf("config changed: %q", data)
	}
}

func TestPackRejectsSymlinkedBundleFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "prompt.md"), []byte("p"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/hosts", filepath.Join(root, "agent.jsonc")); err != nil {
		t.Fatal(err)
	}
	target := BundleTarget{Kind: "agent", Files: func(string) FileSet { return testFiles }, Pack: PackZip}
	if _, _, err := target.pack(root, "id"); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("symlink accepted: %v", err)
	}
}

// E1: a new kind needs one registry entry and no change to the runner, the
// workspace code, or the output code.
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

func TestNoWorkspaceNamesEveryCloneAndCreateCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	err := runTargetAction(&cobra.Command{}, "pull")
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, kind := range []string{"agent", "skill", "workflow", "app"} {
		if !strings.Contains(err.Error(), "major "+kind+" clone") || !strings.Contains(err.Error(), "major "+kind+" create") {
			t.Fatalf("error does not name %s: %v", kind, err)
		}
	}
}

func TestTextOutputIsReadableAndOmitsDownloadURL(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.Flags().Bool("json", false, "")
	result := versionResult{TargetVersion: api.TargetVersion{AgentID: "a1", Name: "Helper", Version: 2}, action: "Pulled"}
	if err := Output(cmd, result); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Pulled Helper (version 2).\n" {
		t.Fatalf("text output: %q", out.String())
	}
	out.Reset()
	_ = cmd.Flags().Set("json", "true")
	if err := Output(cmd, result); err != nil {
		t.Fatal(err)
	}
	if out.String() != "{\"agentId\":\"a1\",\"name\":\"Helper\",\"version\":2}\n" {
		t.Fatalf("json output: %q", out.String())
	}
}

func TestPublishTextListsTriggerFailures(t *testing.T) {
	result := publishResult{&api.TargetPublishResponse{Version: 3, TriggerFailures: []api.TriggerFailure{
		{TriggerID: "t1", ConnectorType: "linear", Events: []string{"issue.created"}, Error: "provider down"},
	}}}
	want := "Published version 3.\nTrigger t1 (linear: issue.created) was not registered: provider down"
	if result.String() != want {
		t.Fatalf("text: %q", result.String())
	}
}
