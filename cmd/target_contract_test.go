package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/major-technology/cli/clients/workspace"
)

const (
	bundleOrgID      = "22222222-2222-4222-8222-222222222222"
	bundleAgentID    = "33333333-3333-4333-8333-333333333333"
	bundleSkillID    = "44444444-4444-4444-8444-444444444444"
	bundleWorkflowID = "55555555-5555-4555-8555-555555555555"
)

// bundleServer fakes the /cli/<kind>s routes and the presigned S3 URLs for one
// bundle kind, and records what the CLI sent.
type bundleServer struct {
	t        *testing.T
	url      string
	apiPath  string
	id       string
	pullBody []byte
	fail     map[string]bool
	invalid  bool

	mu          sync.Mutex
	uploads     [][]byte
	uploadTypes []string
	pushBody    map[string]any
	validate    map[string]any
}

func newBundleServer(t *testing.T, apiPath, id string, pullBody []byte) *bundleServer {
	t.Helper()
	s := &bundleServer{t: t, apiPath: apiPath, id: id, pullBody: pullBody, fail: map[string]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(srv.Close)
	s.url = srv.URL
	restoreAPIClient(t, srv.URL)
	t.Setenv("MAJOR_TOKEN", contractToken)
	return s
}

func (s *bundleServer) idFields(version int) map[string]any {
	switch s.apiPath {
	case "agents":
		return map[string]any{"agentId": s.id, "name": "Helper", "version": version}
	case "skills":
		return map[string]any{"skillId": s.id, "slug": "brand-kit", "version": version}
	default:
		return map[string]any{"workflowId": s.id, "label": "Weekly", "version": version}
	}
}

func (s *bundleServer) serve(w http.ResponseWriter, r *http.Request) {
	prefix := "/" + s.apiPath + "/" + s.id + "/"
	if r.URL.Path == "/s3/get" {
		_, _ = w.Write(s.pullBody)
		return
	}
	if r.URL.Path == "/s3/put" {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.uploads = append(s.uploads, body)
		s.uploadTypes = append(s.uploadTypes, r.Header.Get("Content-Type"))
		s.mu.Unlock()
		return
	}
	if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, prefix) {
		s.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	action := strings.TrimPrefix(r.URL.Path, prefix)
	if s.fail[action] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":{"internal_code":3003,"error_string":"not_org_builder","status_code":403,"message":"denied"}}`)
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	var out map[string]any
	switch action {
	case "pull":
		out = s.idFields(2)
		out["downloadUrl"] = s.url + "/s3/get"
	case "push-upload-url":
		out = map[string]any{"uploadUrl": s.url + "/s3/put", "uploadKey": "upload-key"}
	case "push":
		s.mu.Lock()
		s.pushBody = body
		s.mu.Unlock()
		out = s.idFields(3)
	case "validate":
		s.mu.Lock()
		s.validate = body
		s.mu.Unlock()
		if s.invalid {
			out = map[string]any{"valid": false, "errors": []any{map[string]any{"path": "edges", "message": "cycle"}}, "warnings": []any{}}
		} else {
			out = map[string]any{"valid": true, "errors": []any{}, "warnings": []any{}}
		}
	case "publish":
		if s.apiPath == "workflows" {
			out = map[string]any{"workflowId": s.id, "label": "Weekly", "version": 3, "triggerFailures": []any{
				map[string]any{"triggerId": "t1", "connectorType": "linear", "events": []any{"issue.created"}, "error": "provider down"},
			}}
		} else {
			out = map[string]any{"versionId": "66666666-6666-4666-8666-666666666666", "version": 3}
		}
	default:
		s.t.Errorf("unexpected action %s", action)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for name, content := range files {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(content))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func zipNames(t *testing.T, data []byte) []string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(reader.File))
	for _, f := range reader.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

// bundleWorkspace writes .major/config.json plus files into a temp dir and
// makes it the working directory.
func bundleWorkspace(t *testing.T, target workspace.Target, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: bundleOrgID, Target: target}); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	return dir
}

// runTopLevel runs `major <action>` through the registered top-level command.
func runTopLevel(t *testing.T, action string, flags map[string]string) (string, string, error) {
	t.Helper()
	cmd, _, err := rootCmd.Find([]string{action})
	if err != nil {
		t.Fatal(err)
	}
	flags["json"] = "true"
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set %s: %v", name, err)
		}
		name := name
		t.Cleanup(func() { _ = cmd.Flags().Set(name, cmd.Flags().Lookup(name).DefValue) })
	}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	t.Cleanup(func() {
		cmd.SetOut(nil)
		cmd.SetErr(nil)
	})
	runErr := cmd.RunE(cmd, nil)
	return stdout.String(), stderr.String(), runErr
}

type bundleKind struct {
	name    string
	apiPath string
	id      string
	target  workspace.Target
	files   map[string]string
	pull    []byte
}

func bundleKinds(t *testing.T) []bundleKind {
	return []bundleKind{
		{
			name: "agent", apiPath: "agents", id: bundleAgentID,
			target: workspace.Target{Kind: "agent", AgentID: bundleAgentID},
			files:  map[string]string{"agent.jsonc": "{}", "prompt.md": "You are Helper."},
			pull:   zipBytes(t, map[string]string{"agent.jsonc": "{}", "prompt.md": "pulled"}),
		},
		{
			name: "skill", apiPath: "skills", id: bundleSkillID,
			target: workspace.Target{Kind: "skill", SkillID: bundleSkillID},
			files:  map[string]string{"SKILL.md": "---\nname: brand-kit\ndescription: d\n---\n", "refs/a.md": "a"},
			pull:   zipBytes(t, map[string]string{"SKILL.md": "pulled"}),
		},
		{
			name: "workflow", apiPath: "workflows", id: bundleWorkflowID,
			target: workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID},
			files:  map[string]string{bundleWorkflowID + ".jsonc": `{"label":"Weekly"}`},
			pull:   []byte(`{"label":"Pulled"}`),
		},
	}
}

// B6: every bundle kind and action prints one JSON object with --json.
func TestBundleTargetsPrintOneJSONObjectPerAction(t *testing.T) {
	for _, kind := range bundleKinds(t) {
		t.Run(kind.name, func(t *testing.T) {
			newBundleServer(t, kind.apiPath, kind.id, kind.pull)
			bundleWorkspace(t, kind.target, kind.files)
			for _, action := range []string{"validate", "push", "publish", "pull"} {
				flags := map[string]string{}
				if action == "publish" {
					flags["yes"] = "true"
				}
				stdout, stderr, err := runTopLevel(t, action, flags)
				if err != nil {
					t.Fatalf("%s: %v stderr=%q", action, err, stderr)
				}
				result := decodeOneObject(t, stdout)
				if _, leaked := result["downloadUrl"]; leaked {
					t.Fatalf("%s leaked the presigned URL: %v", action, result)
				}
				assertNoTokenLeak(t, stdout, stderr, err)
			}
		})
	}
}

// B6: nothing reaches stdout after an error, for every bundle kind and action.
func TestBundleTargetErrorsLeaveStdoutEmpty(t *testing.T) {
	for _, kind := range bundleKinds(t) {
		t.Run(kind.name, func(t *testing.T) {
			server := newBundleServer(t, kind.apiPath, kind.id, kind.pull)
			bundleWorkspace(t, kind.target, kind.files)
			for _, action := range []string{"pull", "push", "validate", "publish"} {
				server.fail[action] = true
				server.fail["push-upload-url"] = action == "push"
				flags := map[string]string{}
				if action == "publish" {
					flags["yes"] = "true"
				}
				stdout, _, err := runTopLevel(t, action, flags)
				if err == nil {
					t.Fatalf("%s: expected an error", action)
				}
				if stdout != "" {
					t.Fatalf("%s: stdout after an error: %q", action, stdout)
				}
				server.fail = map[string]bool{}
			}
		})
	}
}

func TestSkillPushPacksBundleFilesOnlyAndValidateUploadsFirst(t *testing.T) {
	server := newBundleServer(t, "skills", bundleSkillID, nil)
	bundleWorkspace(t, workspace.Target{Kind: "skill", SkillID: bundleSkillID}, map[string]string{
		"SKILL.md":                   "---\nname: brand-kit\ndescription: d\n---\n",
		"scripts/run.ts":             "x",
		"tsconfig.json":              "{}",
		".DS_Store":                  "junk",
		"node_modules/pkg/index.js":  "dep",
		"scripts/node_modules/x.js":  "dep",
		"__MACOSX/SKILL.md":          "fork",
		".git/HEAD":                  "ref",
		"scripts/nested/tsconfig.js": "kept",
	})
	if _, _, err := runTopLevel(t, "validate", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if server.validate["uploadKey"] != "upload-key" {
		t.Fatalf("validate body: %v", server.validate)
	}
	if _, _, err := runTopLevel(t, "push", map[string]string{"message": "e2e"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"SKILL.md", "scripts/nested/tsconfig.js", "scripts/run.ts"}
	for i, upload := range server.uploads {
		if got := zipNames(t, upload); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("upload %d entries = %v, want %v", i, got, want)
		}
		if server.uploadTypes[i] != "application/zip" {
			t.Fatalf("upload content type %q", server.uploadTypes[i])
		}
	}
	if server.pushBody["uploadKey"] != "upload-key" || server.pushBody["notes"] != "e2e" {
		t.Fatalf("push body: %v", server.pushBody)
	}
}

func TestSkillPullRemovesStaleBundleFilesAndKeepsSkippedOnes(t *testing.T) {
	newBundleServer(t, "skills", bundleSkillID, zipBytes(t, map[string]string{"SKILL.md": "v2", "refs/new.md": "n"}))
	dir := bundleWorkspace(t, workspace.Target{Kind: "skill", SkillID: bundleSkillID}, map[string]string{
		"SKILL.md":      "v1",
		"refs/old.md":   "stale",
		"tsconfig.json": "{}",
	})
	if _, _, err := runTopLevel(t, "pull", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "refs/old.md")); !os.IsNotExist(err) {
		t.Fatalf("stale file kept: %v", err)
	}
	for rel, want := range map[string]string{"SKILL.md": "v2", "refs/new.md": "n", "tsconfig.json": "{}"} {
		if data, _ := os.ReadFile(filepath.Join(dir, rel)); string(data) != want {
			t.Fatalf("%s = %q, want %q", rel, data, want)
		}
	}
	if _, err := workspace.Load(dir); err != nil {
		t.Fatalf("config lost: %v", err)
	}
}

func TestSkillValidateRejectsSymlinkBeforeUploading(t *testing.T) {
	server := newBundleServer(t, "skills", bundleSkillID, nil)
	dir := bundleWorkspace(t, workspace.Target{Kind: "skill", SkillID: bundleSkillID}, map[string]string{"SKILL.md": "x"})
	if err := os.Symlink("/etc/hosts", filepath.Join(dir, "hosts")); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runTopLevel(t, "validate", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "regular file") || stdout != "" {
		t.Fatalf("symlink validate: err=%v stdout=%q", err, stdout)
	}
	if len(server.uploads) != 0 {
		t.Fatal("uploaded a bundle with a symlink")
	}
}

func TestWorkflowPushSendsRawTextWithoutNotes(t *testing.T) {
	server := newBundleServer(t, "workflows", bundleWorkflowID, nil)
	definition := `{"label":"Weekly"}`
	bundleWorkspace(t, workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID}, map[string]string{bundleWorkflowID + ".jsonc": definition})
	if _, _, err := runTopLevel(t, "push", map[string]string{"message": "ignored"}); err != nil {
		t.Fatal(err)
	}
	if len(server.uploads) != 1 || string(server.uploads[0]) != definition || server.uploadTypes[0] != "text/plain" {
		t.Fatalf("uploads: %q types=%v", server.uploads, server.uploadTypes)
	}
	if _, sent := server.pushBody["notes"]; sent || server.pushBody["uploadKey"] != "upload-key" {
		t.Fatalf("push body: %v", server.pushBody)
	}
	if _, _, err := runTopLevel(t, "validate", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if server.validate["definitionText"] != definition {
		t.Fatalf("validate body: %v", server.validate)
	}
}

func TestWorkflowValidateFailureNamesThePath(t *testing.T) {
	server := newBundleServer(t, "workflows", bundleWorkflowID, nil)
	server.invalid = true
	bundleWorkspace(t, workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID}, map[string]string{bundleWorkflowID + ".jsonc": "{}"})
	stdout, _, err := runTopLevel(t, "validate", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "edges: cycle") || stdout != "" {
		t.Fatalf("invalid workflow: err=%v stdout=%q", err, stdout)
	}
}

// B5: workflow publish returns triggerFailures, and the CLI prints them.
func TestWorkflowPublishPrintsTriggerFailures(t *testing.T) {
	newBundleServer(t, "workflows", bundleWorkflowID, nil)
	bundleWorkspace(t, workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID}, map[string]string{bundleWorkflowID + ".jsonc": "{}"})
	stdout, _, err := runTopLevel(t, "publish", map[string]string{"yes": "true"})
	if err != nil {
		t.Fatal(err)
	}
	result := decodeOneObject(t, stdout)
	failures, _ := result["triggerFailures"].([]any)
	if len(failures) != 1 {
		t.Fatalf("triggerFailures: %v", result)
	}

	cmd, _, _ := rootCmd.Find([]string{"publish"})
	var text bytes.Buffer
	cmd.SetOut(&text)
	t.Cleanup(func() { cmd.SetOut(nil) })
	_ = cmd.Flags().Set("yes", "true")
	_ = cmd.Flags().Set("json", "false")
	t.Cleanup(func() { _ = cmd.Flags().Set("yes", "false") })
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "Trigger t1 (linear: issue.created) was not registered: provider down") {
		t.Fatalf("text output: %q", text.String())
	}
}

// --slug is registered by the app kind only; a bundle kind rejects it.
func TestPublishSlugFlagIsAppOnly(t *testing.T) {
	newBundleServer(t, "skills", bundleSkillID, nil)
	bundleWorkspace(t, workspace.Target{Kind: "skill", SkillID: bundleSkillID}, map[string]string{"SKILL.md": "x"})
	stdout, _, err := runTopLevel(t, "publish", map[string]string{"yes": "true", "slug": "my-app"})
	if err == nil || !strings.Contains(err.Error(), "--slug is only for app workspaces") || stdout != "" {
		t.Fatalf("skill publish --slug: err=%v stdout=%q", err, stdout)
	}
	for _, action := range []string{"pull", "push", "validate"} {
		cmd, _, _ := rootCmd.Find([]string{action})
		if cmd.Flags().Lookup("slug") != nil {
			t.Fatalf("major %s must not take --slug", action)
		}
	}
}
