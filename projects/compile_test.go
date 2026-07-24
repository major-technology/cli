package projects

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileValidProject(t *testing.T) {
	res, issues := Compile("testdata/valid")
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	if res.Config.ConfigVersion != 1 {
		t.Fatalf("configVersion = %d, want 1", res.Config.ConfigVersion)
	}
	if res.Config.Project.Name != "Valid Project" {
		t.Fatalf("project name = %q", res.Config.Project.Name)
	}
	if len(res.Config.Agents) != 2 {
		t.Fatalf("agents = %d, want 2", len(res.Config.Agents))
	}

	triage := res.Config.Agents[0]
	if triage.Slug != "triage" {
		t.Fatalf("first agent = %q, want triage (sorted)", triage.Slug)
	}
	if !strings.Contains(triage.SystemPrompt, "support triage agent") {
		t.Fatalf("prompt file not inlined: %q", triage.SystemPrompt)
	}

	// Round-trips as JSON and matches the marshaled config.
	var back CompiledConfig
	if err := json.Unmarshal(res.ConfigJSON, &back); err != nil {
		t.Fatalf("ConfigJSON does not round-trip: %v", err)
	}
	if len(res.Hash) != 64 {
		t.Fatalf("hash length = %d, want 64 hex chars", len(res.Hash))
	}
}

func TestCompileIsDeterministic(t *testing.T) {
	a, issues := Compile("testdata/valid")
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	b, _ := Compile("testdata/valid")
	if a.Hash != b.Hash {
		t.Fatalf("hashes differ across runs: %s vs %s", a.Hash, b.Hash)
	}
}

// TestCompileRelativeDir checks that a relative --dir (the CLI default of
// ".") compiles cleanly. filepath.Clean(".") stays ".", but joining a
// non-empty agent dir onto "." and cleaning the result strips the leading
// "./", so comparing the two literally rejected every legitimate prompt file.
// Compile must resolve dir to an absolute path before that comparison.
func TestCompileRelativeDir(t *testing.T) {
	t.Chdir("testdata/valid")

	res, issues := Compile(".")
	if len(issues) != 0 {
		t.Fatalf("expected no issues compiling from a relative dir, got: %+v", issues)
	}

	triage := res.Config.Agents[0]
	if triage.Slug != "triage" {
		t.Fatalf("first agent = %q, want triage (sorted)", triage.Slug)
	}
	if !strings.Contains(triage.SystemPrompt, "support triage agent") {
		t.Fatalf("prompt file not inlined: %q", triage.SystemPrompt)
	}
}

func TestCompilePromptEscapeRejected(t *testing.T) {
	_, issues := Compile("testdata/escape-prompt")
	if !findIssue(issues, "outside the project directory") {
		t.Fatalf("expected containment issue, got: %+v", issues)
	}
}

// TestCompileRelativeDirEscapeRejected checks that resolving dir to an
// absolute path does not weaken containment: a "../"-escaping prompt ref must
// still be rejected when --dir is "." itself, the same shape that exposed the
// original bug.
func TestCompileRelativeDirEscapeRejected(t *testing.T) {
	t.Chdir("testdata/escape-prompt")

	_, issues := Compile(".")
	if !findIssue(issues, "outside the project directory") {
		t.Fatalf("expected containment issue, got: %+v", issues)
	}
}

func TestCompileSymlinkPromptRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	target := filepath.Join(dir, "real-prompt.md")
	if err := os.WriteFile(target, []byte("real"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "src", "agents", "linked", "prompt.md")
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	agentJSON := `{"slug":"linked","name":"Linked","systemPrompt":{"file":"./prompt.md"}}`
	if err := os.WriteFile(filepath.Join(dir, "src", "agents", "linked", "agent.json"), []byte(agentJSON), 0644); err != nil {
		t.Fatal(err)
	}

	_, issues := Compile(dir)
	if !findIssue(issues, "symlink") {
		t.Fatalf("expected symlink issue, got: %+v", issues)
	}
}

func TestCompileOversizePromptRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	agentDir := filepath.Join(dir, "src", "agents", "big")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, MaxPromptFileBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(agentDir, "prompt.md"), big, 0644); err != nil {
		t.Fatal(err)
	}
	agentJSON := `{"slug":"big","name":"Big","systemPrompt":{"file":"./prompt.md"}}`
	if err := os.WriteFile(filepath.Join(agentDir, "agent.json"), []byte(agentJSON), 0644); err != nil {
		t.Fatal(err)
	}

	_, issues := Compile(dir)
	if !findIssue(issues, "200KB") {
		t.Fatalf("expected size cap issue, got: %+v", issues)
	}
}

func TestCompileMissingPromptFile(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	agentDir := filepath.Join(dir, "src", "agents", "ghost")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	agentJSON := `{"slug":"ghost","name":"Ghost","systemPrompt":{"file":"./nope.md"}}`
	if err := os.WriteFile(filepath.Join(agentDir, "agent.json"), []byte(agentJSON), 0644); err != nil {
		t.Fatal(err)
	}

	_, issues := Compile(dir)
	if !findIssue(issues, "not found") {
		t.Fatalf("expected missing-file issue, got: %+v", issues)
	}
}

// writeFixtureProject writes a minimal valid project.json into dir.
func writeFixtureProject(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{"name":"Temp"}`), 0644); err != nil {
		t.Fatal(err)
	}
}
