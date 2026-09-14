package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testTargetID    = "11111111-1111-4111-8111-111111111111"
	testOrgID       = "22222222-2222-4222-8222-222222222222"
	betterAuthOrgID = "shjRtTSq4A9wNTMFpcB7VKBYPR5spmYm"
)

func TestTargetValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target Target
		valid  bool
	}{
		{"app", Target{Kind: "app", ApplicationID: "11111111-1111-4111-8111-111111111111"}, true},
		{"skill", Target{Kind: "skill", SkillID: "11111111-1111-4111-8111-111111111111"}, true},
		{"agent", Target{Kind: "agent", AgentID: "11111111-1111-4111-8111-111111111111"}, true},
		{"workflow", Target{Kind: "workflow", WorkflowID: "11111111-1111-4111-8111-111111111111"}, true},
		{"wrong-id", Target{Kind: "skill", ApplicationID: "11111111-1111-4111-8111-111111111111"}, false},
		{"unknown", Target{Kind: "project", ApplicationID: "11111111-1111-4111-8111-111111111111"}, false},
		{"invalid-id", Target{Kind: "app", ApplicationID: "not-a-uuid"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.target.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate() = %v; valid=%v", err, tc.valid)
			}
		})
	}
}

func TestConfigAcceptsBetterAuthAndUUIDOrganizationID(t *testing.T) {
	for _, orgID := range []string{betterAuthOrgID, testOrgID} {
		cfg := validAppConfig()
		cfg.OrganizationID = orgID
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate() rejected organizationId %q: %v", orgID, err)
		}
	}
}

func TestConfigRejectsEmptyOrganizationID(t *testing.T) {
	cfg := validAppConfig()
	cfg.OrganizationID = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty organizationId")
	}
}

func TestWriteRejectsNullExistingConfig(t *testing.T) {
	dir := t.TempDir()
	writeRawConfig(t, dir, "null")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Write panicked on null config: %v", r)
		}
	}()
	err := Write(dir, validAppConfig())
	if err == nil {
		t.Fatal("expected error writing over null config")
	}
	configPath := filepath.Join(dir, ".major", "config.json")
	if !strings.Contains(err.Error(), configPath) {
		t.Fatalf("error %q should include path %q", err, configPath)
	}
	if !strings.Contains(err.Error(), "repair .major/config.json") {
		t.Fatalf("error %q should include repair guidance", err)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := validAppConfig()
	if err := Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	assertConfigEqual(t, *got, cfg)
}

func TestWriteReadBetterAuthOrganizationID(t *testing.T) {
	dir := t.TempDir()
	cfg := validAppConfig()
	cfg.OrganizationID = betterAuthOrgID
	if err := Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	assertConfigEqual(t, *got, cfg)
}

func TestLoadFromNestedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "src", "app")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, validAppConfig()); err != nil {
		t.Fatal(err)
	}
	got, err := Load(nested)
	if err != nil {
		t.Fatal(err)
	}
	assertConfigEqual(t, *got, validAppConfig())
}

func TestLoadNonGitDirectory(t *testing.T) {
	dir := isolatedTree(t)
	if err := Write(dir, validSkillConfig()); err != nil {
		t.Fatal(err)
	}
	got, err := Load(filepath.Join(dir, "nested"))
	if err != nil {
		t.Fatal(err)
	}
	assertConfigEqual(t, *got, validSkillConfig())
}

func TestLoadNearestConfig(t *testing.T) {
	dir := isolatedTree(t)
	inner := filepath.Join(dir, "skill")
	if err := os.MkdirAll(inner, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, validAppConfig()); err != nil {
		t.Fatal(err)
	}
	if err := Write(inner, validSkillConfig()); err != nil {
		t.Fatal(err)
	}
	got, err := Load(inner)
	if err != nil {
		t.Fatal(err)
	}
	assertConfigEqual(t, *got, validSkillConfig())
}

func TestLoadStopsAtNestedGitDirectory(t *testing.T) {
	outer := isolatedTree(t)
	if err := Write(outer, validAppConfig()); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "nested-repo")
	if err := os.MkdirAll(filepath.Join(inner, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(inner, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := Load(filepath.Join(inner, "src"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() = %v, want ErrNotFound", err)
	}
}

func TestLoadStopsAtNestedGitWorktreeFile(t *testing.T) {
	outer := isolatedTree(t)
	if err := Write(outer, validAppConfig()); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "worktree")
	if err := os.MkdirAll(filepath.Join(inner, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, ".git"), []byte("gitdir: /tmp/fake-worktree\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(filepath.Join(inner, "src"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() = %v, want ErrNotFound", err)
	}
}

func TestLoadMalformedFile(t *testing.T) {
	dir := isolatedTree(t)
	path := filepath.Join(dir, ".major")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(path, "config.json")
	if err := os.WriteFile(configPath, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for malformed config")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("malformed file must not return ErrNotFound")
	}
	if !strings.Contains(err.Error(), configPath) {
		t.Fatalf("error %q should include path %q", err, configPath)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := isolatedTree(t)
	_, err := Load(filepath.Join(dir, "nested"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() = %v, want ErrNotFound", err)
	}
}

func TestLoadTwoTargetIDs(t *testing.T) {
	dir := isolatedTree(t)
	writeRawConfig(t, dir, `{
		"organizationId": "`+testOrgID+`",
		"target": {
			"kind": "app",
			"applicationId": "`+testTargetID+`",
			"skillId": "`+testTargetID+`"
		}
	}`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for two target IDs")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("invalid target must not return ErrNotFound")
	}
}

func TestLoadMissingOrganization(t *testing.T) {
	dir := isolatedTree(t)
	writeRawConfig(t, dir, `{
		"target": {
			"kind": "app",
			"applicationId": "`+testTargetID+`"
		}
	}`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing organization")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("missing organization must not return ErrNotFound")
	}
}

func TestWritePreservesUnknownFields(t *testing.T) {
	dir := t.TempDir()
	writeRawConfig(t, dir, `{
		"organizationId": "`+testOrgID+`",
		"target": {
			"kind": "app",
			"applicationId": "`+testTargetID+`"
		},
		"comment": "keep me",
		"extra": {"nested": true}
	}`)
	cfg := validAppConfig()
	cfg.OrganizationID = "33333333-3333-4333-8333-333333333333"
	if err := Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".major", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if string(parsed["comment"]) != `"keep me"` {
		t.Fatalf("comment = %s, want preserved", parsed["comment"])
	}
	if !bytes.Contains(parsed["extra"], []byte(`"nested"`)) {
		t.Fatalf("extra = %s, want nested object preserved", parsed["extra"])
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.OrganizationID != cfg.OrganizationID {
		t.Fatalf("organizationId = %q, want %q", got.OrganizationID, cfg.OrganizationID)
	}
}

func TestLoadDoesNotFallBackPastNonAppTarget(t *testing.T) {
	dir := isolatedTree(t)
	inner := filepath.Join(dir, "skill")
	if err := os.MkdirAll(inner, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, validAppConfig()); err != nil {
		t.Fatal(err)
	}
	writeRawConfig(t, inner, `{
		"organizationId": "`+testOrgID+`",
		"target": {
			"kind": "skill",
			"applicationId": "`+testTargetID+`"
		}
	}`)
	_, err := Load(inner)
	if err == nil {
		t.Fatal("expected error for mismatched non-app target, not parent fallback")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("invalid non-app target must not return ErrNotFound")
	}
}

func TestWriteDoesNotRequireGit(t *testing.T) {
	dir := t.TempDir()
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("temp dir unexpectedly has .git: %v", err)
	}
	if err := Write(dir, validWorkflowConfig()); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	assertConfigEqual(t, *got, validWorkflowConfig())
}

func TestIgnoreLocalConfigGitExclude(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	if err := Write(dir, validAppConfig()); err != nil {
		t.Fatal(err)
	}
	if err := IgnoreLocalConfig(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".major", "other.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if code := gitCheckIgnore(t, dir, ".major/config.json"); code != 0 {
		t.Fatalf("git check-ignore .major/config.json exited %d, want 0", code)
	}
	if code := gitCheckIgnore(t, dir, ".major/other.json"); code != 1 {
		t.Fatalf("git check-ignore .major/other.json exited %d, want 1", code)
	}
	if err := IgnoreLocalConfig(dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(excludePath(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "/.major/config.json") != 1 {
		t.Fatalf("exclude rules duplicated:\n%s", raw)
	}
}

func TestIgnoreLocalConfigNonGitNoop(t *testing.T) {
	dir := t.TempDir()
	if err := IgnoreLocalConfig(dir); err != nil {
		t.Fatalf("non-git ignore should be no-op, got %v", err)
	}
	if err := Write(dir, validAgentConfig()); err != nil {
		t.Fatal(err)
	}
}

func validAppConfig() Config {
	return Config{
		OrganizationID: testOrgID,
		Target:         Target{Kind: "app", ApplicationID: testTargetID},
	}
}

func validSkillConfig() Config {
	return Config{
		OrganizationID: testOrgID,
		Target:         Target{Kind: "skill", SkillID: testTargetID},
	}
}

func validAgentConfig() Config {
	return Config{
		OrganizationID: testOrgID,
		Target:         Target{Kind: "agent", AgentID: testTargetID},
	}
}

func validWorkflowConfig() Config {
	return Config{
		OrganizationID: testOrgID,
		Target:         Target{Kind: "workflow", WorkflowID: testTargetID},
	}
}

func assertConfigEqual(t *testing.T, got, want Config) {
	t.Helper()
	if got.OrganizationID != want.OrganizationID {
		t.Fatalf("organizationId = %q, want %q", got.OrganizationID, want.OrganizationID)
	}
	if got.Target != want.Target {
		t.Fatalf("target = %+v, want %+v", got.Target, want.Target)
	}
}

func isolatedTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeRawConfig(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, ".major")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "config.json"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitCheckIgnore(t *testing.T, dir, path string) int {
	t.Helper()
	cmd := exec.Command("git", "check-ignore", "-q", path)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	t.Fatalf("git check-ignore %s: %v", path, err)
	return -1
}

func excludePath(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--git-path", "info/exclude")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	p := strings.TrimSpace(string(out))
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return p
}
