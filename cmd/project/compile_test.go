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
//
// Exit code contract: mono-builder's compile-job wrapper shells out to
// `major project compile --json` and classifies its exit code - EXACTLY 1
// means "content failure" (validation issues found), any other non-zero
// means "infra failure" (crash/panic). This test drives newCompileCmd()
// in-process via cobra, so no OS process actually exits here; the real
// chain is: this RunE returns a non-nil error -> cobra bubbles it up as
// rootCmd.Execute()'s return value -> cmd/root.go's Execute() (called from
// main.main) sees err != nil -> os.Exit(1). That os.Exit(1) call is the
// ONLY os.Exit in this codebase (confirmed by `grep -rn "os.Exit" cmd/`)
// and it is unconditional - there is no switch on error type/value that
// could produce a different code. So "RunE returned a non-nil error" and
// "the binary exits 1" are the same fact observed at different seams; we
// assert the former (plus the specific validation-issue error, so this
// test fails loudly if some unrelated error - e.g. a bad --dir - started
// satisfying it instead) as the closest in-process proxy for the latter.
func TestCompileJSONContractFailure(t *testing.T) {
	dir := writeInvalidProject(t)

	stdout, stderr, err := runCommandSplit(t, newCompileCmd(), "--dir", dir, "--json")
	if err == nil {
		t.Fatalf("expected the validation-issue error that maps to exit code 1, got nil, stdout: %s", stdout)
	}
	if !strings.Contains(err.Error(), "validation issue") {
		t.Fatalf("expected a validation-issue error (the one path that maps to exit code 1), got: %v", err)
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
