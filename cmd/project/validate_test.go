package project

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func writeValidProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{"name":"T"}`), 0644); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(dir, "src", "agents", "a")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent.json"), []byte(`{"slug":"a","name":"A","systemPrompt":"hi"}`), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func runCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestValidateCommandValidProject(t *testing.T) {
	dir := writeValidProject(t)
	out, err := runCommand(t, newValidateCmd(), "--dir", dir)
	if err != nil {
		t.Fatalf("expected success, got %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "valid") {
		t.Fatalf("expected success message, got: %s", out)
	}
}

func TestValidateCommandInvalidProject(t *testing.T) {
	dir := t.TempDir() // no project.json
	out, err := runCommand(t, newValidateCmd(), "--dir", dir)
	if err == nil {
		t.Fatalf("expected error exit, output: %s", out)
	}
	if !strings.Contains(out, "project.json") {
		t.Fatalf("expected issue mentioning project.json, got: %s", out)
	}
}

func TestCompileCommandOutputsJSON(t *testing.T) {
	dir := writeValidProject(t)
	out, err := runCommand(t, newCompileCmd(), "--dir", dir)
	if err != nil {
		t.Fatalf("expected success, got %v (output: %s)", err, out)
	}
	if !strings.Contains(out, `"configVersion":1`) && !strings.Contains(out, `"configVersion": 1`) {
		t.Fatalf("expected compiled config JSON, got: %s", out)
	}
}
