package project

import (
	"bytes"
	"encoding/json"
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

// runCommandSplit captures stdout and stderr in separate buffers, so tests
// can assert which stream machine-readable output actually lands on.
func runCommandSplit(t *testing.T, cmd *cobra.Command, args ...string) (stdout string, stderr string, err error) {
	t.Helper()
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
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

func TestValidateCommandJSONGoesToStdout(t *testing.T) {
	dir := writeValidProject(t)
	stdout, _, err := runCommandSplit(t, newValidateCmd(), "--dir", dir, "--json")
	if err != nil {
		t.Fatalf("expected success, got %v (stdout: %s)", err, stdout)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not parseable JSON: %v (stdout: %q)", err, stdout)
	}
	if payload["valid"] != true {
		t.Fatalf("expected valid:true, got: %s", stdout)
	}
}

func TestValidateCommandInvalidJSONGoesToStdout(t *testing.T) {
	dir := t.TempDir() // no project.json
	stdout, _, err := runCommandSplit(t, newValidateCmd(), "--dir", dir, "--json")
	if err == nil {
		t.Fatalf("expected error exit, stdout: %s", stdout)
	}

	// Decode (rather than Unmarshal) the leading value only: this bare,
	// parent-less *cobra.Command appends its own "Usage: ..." text after our
	// write when RunE errors (cobra treats it as its own root here). The real
	// CLI never hits that path - project.Cmd is always mounted under rootCmd,
	// which sets SilenceUsage/SilenceErrors - so this is a standalone-test
	// artifact, not something the JSON consumer sees in production.
	var payload map[string]any
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&payload); err != nil {
		t.Fatalf("stdout does not start with parseable JSON: %v (stdout: %q)", err, stdout)
	}
	if payload["valid"] != false {
		t.Fatalf("expected valid:false, got: %s", stdout)
	}
}
