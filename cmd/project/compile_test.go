package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeInvalidProject writes a project.json plus one agent.json missing
// required fields (name, systemPrompt), so Compile reports issues and
// produces no config.
func writeInvalidProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{"name":"T"}`), 0644); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(dir, "src", "agents", "bad")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent.json"), []byte(`{"slug":"bad"}`), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

// TestCompileJSONContractSuccess pins the success half of `major project
// compile --json`: mono-builder's compile job (which replaced the CLI's own
// compile-and-report harness) shells out to this exact command and parses
// stdout directly. It must be exit 0, exactly one line of valid JSON on
// stdout carrying the canonical config, and nothing on stderr.
func TestCompileJSONContractSuccess(t *testing.T) {
	dir := writeValidProject(t)

	stdout, stderr, err := runCommandSplit(t, newCompileCmd(), "--dir", dir, "--json")
	if err != nil {
		t.Fatalf("expected success, got %v (stdout: %s)", err, stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}

	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one line on stdout, got %d: %q", len(lines), stdout)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("stdout line is not valid JSON: %v (line: %q)", err, lines[0])
	}
	if payload["configVersion"] != float64(1) {
		t.Fatalf("expected configVersion 1, got: %v", payload["configVersion"])
	}
}

// TestCompileJSONContractFailure pins the failure half of the same contract:
// mono-builder's compile job treats "no parseable JSON on stdout" as the
// failure signal and surfaces stderr's issue text to the user. Stdout must
// never carry a JSON error/report object - only the canonical config, and
// only on success.
func TestCompileJSONContractFailure(t *testing.T) {
	dir := writeInvalidProject(t)

	stdout, stderr, err := runCommandSplit(t, newCompileCmd(), "--dir", dir, "--json")
	if err == nil {
		t.Fatalf("expected non-zero exit, stdout: %s", stdout)
	}

	// Decode (rather than split-by-line) because a leaked report object
	// would be pretty-printed across several lines - a per-line parse check
	// would miss it entirely. Any successfully decoded value, spanning one
	// line or many, is a contract violation.
	var probe json.RawMessage
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&probe); err == nil {
		t.Fatalf("stdout contains a parseable JSON value, want none: %q (parsed: %s)", stdout, probe)
	}

	if stderr == "" {
		t.Fatal("expected issue text on stderr, got none")
	}
}
