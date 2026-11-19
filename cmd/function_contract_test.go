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
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/major-technology/cli/clients/workspace"
)

var functionSources = map[string]string{
	"index.ts":      "import { greeting } from \"./lib\";\nexport default async function () {\n\treturn { message: greeting };\n}\n",
	"lib.ts":        "export const greeting = \"inlined-greeting\";\n",
	"function.json": `{"label":"Hello"}`,
}

func zipEntry(t *testing.T, data []byte, name string) string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range reader.File {
		if f.Name == name {
			in, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()
			content, _ := io.ReadAll(in)
			return string(content)
		}
	}
	t.Fatalf("no %s in archive: %v", name, zipNames(t, data))
	return ""
}

func TestFunctionPushUploadsSourcesWithTheBuiltBundle(t *testing.T) {
	server := newBundleServer(t, "functions", bundleFunctionID, nil)
	files := map[string]string{"node_modules/dep/index.js": "dep", ".build/stale.js": "stale"}
	for name, content := range functionSources {
		files[name] = content
	}
	dir := bundleWorkspace(t, workspace.Target{Kind: "function", FunctionID: bundleFunctionID}, files)

	if _, stderr, err := runTopLevel(t, "push", map[string]string{}); err != nil {
		t.Fatalf("push: %v stderr=%q", err, stderr)
	}
	if len(server.uploads) != 1 || server.uploadTypes[0] != "application/zip" {
		t.Fatalf("uploads: %d types=%v", len(server.uploads), server.uploadTypes)
	}
	want := []string{".build/bundle.js", "function.json", "index.ts", "lib.ts"}
	if got := zipNames(t, server.uploads[0]); !reflect.DeepEqual(got, want) {
		t.Fatalf("archive entries: %v, want %v", got, want)
	}
	built := zipEntry(t, server.uploads[0], ".build/bundle.js")
	if !strings.Contains(built, "inlined-greeting") || !strings.Contains(built, "Hello") {
		t.Fatalf("bundle does not inline the import and function.json: %q", built)
	}
	if _, err := os.Stat(filepath.Join(dir, ".build", "bundle.js")); !os.IsNotExist(err) {
		t.Fatalf("the built bundle was written to the workspace: %v", err)
	}
}

func TestFunctionThatDoesNotBuildIsAnInvalidBundleAndUploadsNothing(t *testing.T) {
	server := newBundleServer(t, "functions", bundleFunctionID, nil)
	bundleWorkspace(t, workspace.Target{Kind: "function", FunctionID: bundleFunctionID}, map[string]string{
		"index.ts":      "import { missing } from \"./nowhere\";\nexport default async function () {\n\treturn missing;\n}\n",
		"function.json": "{}",
	})
	for _, action := range []string{"validate", "push"} {
		stdout, stderr, err := runTopLevel(t, action, map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "index.ts:1") || stdout != "" {
			t.Fatalf("%s: err=%v stdout=%q", action, err, stdout)
		}
		if !strings.Contains(stderr, `"invalid":true`) {
			t.Fatalf("%s: stderr does not mark the bundle invalid: %q", action, stderr)
		}
	}
	if len(server.uploads) != 0 {
		t.Fatalf("uploaded %d bundles of a function that does not build", len(server.uploads))
	}
}

func TestFunctionPullKeepsSourcesAndDropsTheBuiltBundle(t *testing.T) {
	pull := zipBytes(t, map[string]string{"index.ts": "pulled", "function.json": "{}", ".build/bundle.js": "built"})
	newBundleServer(t, "functions", bundleFunctionID, pull)
	dir := bundleWorkspace(t, workspace.Target{Kind: "function", FunctionID: bundleFunctionID}, functionSources)
	if _, stderr, err := runTopLevel(t, "pull", map[string]string{}); err != nil {
		t.Fatalf("pull: %v stderr=%q", err, stderr)
	}
	if content, _ := os.ReadFile(filepath.Join(dir, "index.ts")); string(content) != "pulled" {
		t.Fatalf("index.ts: %q", content)
	}
	for _, gone := range []string{"lib.ts", ".build/bundle.js"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(gone))); !os.IsNotExist(err) {
			t.Fatalf("%s should not be in the workspace after a pull: %v", gone, err)
		}
	}
}

// workflowServer fakes a workflow and its functions: the /workflows and
// /functions routes and their presigned URLs, recording requests in order.
type workflowServer struct {
	t       *testing.T
	url     string
	archive []byte
	fail    map[string]bool

	mu      sync.Mutex
	calls   []string
	uploads map[string][]byte
	bodies  map[string]map[string]any
}

func newWorkflowServer(t *testing.T) *workflowServer {
	t.Helper()
	s := &workflowServer{t: t, fail: map[string]bool{}, uploads: map[string][]byte{}, bodies: map[string]map[string]any{}}
	srv := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(srv.Close)
	s.url = srv.URL
	restoreAPIClient(t, srv.URL)
	t.Setenv("MAJOR_TOKEN", contractToken)
	return s
}

func (s *workflowServer) called(prefix string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var found []string
	for _, call := range s.calls {
		if strings.HasPrefix(call, prefix) {
			found = append(found, call)
		}
	}
	return found
}

func (s *workflowServer) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if strings.HasPrefix(r.URL.Path, "/s3/put/") {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.uploads[strings.TrimPrefix(r.URL.Path, "/s3/put/")] = body
		s.mu.Unlock()
		return
	}
	switch r.URL.Path {
	case "/s3/get/definition":
		_, _ = w.Write([]byte(`{"label":"Pulled"}`))
		return
	case "/s3/get/archive":
		_, _ = w.Write(s.archive)
		return
	}
	call := strings.TrimPrefix(r.URL.Path, "/")
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	s.mu.Lock()
	s.calls = append(s.calls, call)
	s.bodies[call] = body
	s.mu.Unlock()
	if s.fail[call] {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":{"internal_code":3003,"error_string":"denied","status_code":403,"message":"denied"}}`)
		return
	}
	workflow := "workflows/" + bundleWorkflowID + "/"
	function := "functions/" + bundleFunctionID + "/"
	var out map[string]any
	switch call {
	case "functions":
		out = map[string]any{"functionId": bundleFunctionID, "slug": body["slug"], "version": nil}
	case workflow + "pull":
		out = map[string]any{"workflowId": bundleWorkflowID, "label": "Weekly", "version": 4, "downloadUrl": s.url + "/s3/get/definition"}
		if s.archive != nil {
			out["functionsDownloadUrl"] = s.url + "/s3/get/archive"
		}
	case workflow + "push-upload-url":
		out = map[string]any{
			"uploadUrl": s.url + "/s3/put/definition", "uploadKey": "definition-key",
			"functionsUploadUrl": s.url + "/s3/put/archive", "functionsUploadKey": "archive-key",
		}
	case workflow + "push", workflow + "publish":
		out = map[string]any{"workflowId": bundleWorkflowID, "label": "Weekly", "version": 5}
	case workflow + "validate", function + "validate":
		out = map[string]any{"valid": true, "errors": []any{}, "warnings": []any{}}
	case function + "push-upload-url":
		out = map[string]any{"uploadUrl": s.url + "/s3/put/function", "uploadKey": "function-key"}
	case function + "push", function + "publish":
		out = map[string]any{"functionId": bundleFunctionID, "slug": "hello", "version": 2}
	default:
		s.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(out)
}

// workflowWithFunction is a workflow workspace holding one function workspace.
func workflowWithFunction(t *testing.T) string {
	t.Helper()
	files := map[string]string{
		bundleWorkflowID + ".jsonc":                 `{"label":"Weekly"}`,
		"functions/hello/node_modules/dep/index.js": "dep",
	}
	for name, content := range functionSources {
		files["functions/hello/"+name] = content
	}
	dir := bundleWorkspace(t, workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID}, files)
	nested := workspace.Config{OrganizationID: bundleOrgID, Target: workspace.Target{Kind: "function", FunctionID: bundleFunctionID}}
	if err := workspace.Write(filepath.Join(dir, "functions", "hello"), nested); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestWorkflowActionsRunOnEveryFunctionBeforeTheWorkflow(t *testing.T) {
	server := newWorkflowServer(t)
	workflowWithFunction(t)
	workflow := "workflows/" + bundleWorkflowID + "/"
	function := "functions/" + bundleFunctionID + "/"

	for _, action := range []string{"validate", "push", "publish"} {
		flags := map[string]string{}
		if action == "publish" {
			flags["yes"] = "true"
		}
		stdout, stderr, err := runTopLevel(t, action, flags)
		if err != nil {
			t.Fatalf("%s: %v stderr=%q", action, err, stderr)
		}
		calls := server.called("")
		last := len(calls) - 1
		functionAt, workflowAt := -1, -1
		for i, call := range calls {
			if call == function+action {
				functionAt = i
			}
			if call == workflow+action {
				workflowAt = i
			}
		}
		if functionAt < 0 || workflowAt != last || functionAt > workflowAt {
			t.Fatalf("%s: the function must go first, then the workflow: %v", action, calls)
		}
		result := decodeOneObject(t, stdout)
		nested, _ := result["nested"].([]any)
		if len(nested) != 1 {
			t.Fatalf("%s: nested results: %v", action, result)
		}
		item := nested[0].(map[string]any)
		if item["kind"] != "function" || item["id"] != bundleFunctionID || item["path"] != "functions/hello" {
			t.Fatalf("%s: nested item: %v", action, item)
		}
	}
	if server.bodies[workflow+"push"]["functionsUploadKey"] != "archive-key" {
		t.Fatalf("workflow push body: %v", server.bodies[workflow+"push"])
	}
	if string(server.uploads["definition"]) != `{"label":"Weekly"}` {
		t.Fatalf("definition upload: %q", server.uploads["definition"])
	}
	want := []string{"functions/hello/.major/config.json", "functions/hello/function.json", "functions/hello/index.ts", "functions/hello/lib.ts"}
	if got := zipNames(t, server.uploads["archive"]); !reflect.DeepEqual(got, want) {
		t.Fatalf("workflow archive entries: %v, want %v", got, want)
	}
}

func TestWorkflowPushStopsWhenAFunctionFails(t *testing.T) {
	server := newWorkflowServer(t)
	workflowWithFunction(t)
	server.fail["functions/"+bundleFunctionID+"/push"] = true
	stdout, _, err := runTopLevel(t, "push", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "functions/hello") || stdout != "" {
		t.Fatalf("push: err=%v stdout=%q", err, stdout)
	}
	if calls := server.called("workflows/"); len(calls) != 0 {
		t.Fatalf("the workflow was touched after a function failed: %v", calls)
	}
}

func TestWorkflowPushRejectsAFunctionsFolderThatIsNotAWorkspace(t *testing.T) {
	server := newWorkflowServer(t)
	bundleWorkspace(t, workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID}, map[string]string{
		bundleWorkflowID + ".jsonc": "{}",
		"functions/bare/index.ts":   "export default async function () {}\n",
	})
	stdout, _, err := runTopLevel(t, "push", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "major function create functions/bare") || stdout != "" {
		t.Fatalf("push: err=%v stdout=%q", err, stdout)
	}
	if calls := server.called(""); len(calls) != 0 {
		t.Fatalf("requests were made: %v", calls)
	}
}

func TestWorkflowPullRestoresFunctionWorkspacesFromTheArchive(t *testing.T) {
	server := newWorkflowServer(t)
	nested, _ := json.Marshal(workspace.Config{OrganizationID: bundleOrgID, Target: workspace.Target{Kind: "function", FunctionID: bundleFunctionID}})
	server.archive = zipBytes(t, map[string]string{
		"functions/hello/index.ts":           "pulled",
		"functions/hello/function.json":      "{}",
		"functions/hello/.major/config.json": string(nested),
	})
	dir := bundleWorkspace(t, workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID}, map[string]string{
		bundleWorkflowID + ".jsonc":             "{}",
		"functions/removed/index.ts":            "stale",
		"functions/hello/node_modules/dep/a.js": "kept",
	})
	if _, stderr, err := runTopLevel(t, "pull", map[string]string{}); err != nil {
		t.Fatalf("pull: %v stderr=%q", err, stderr)
	}
	if definition, _ := os.ReadFile(filepath.Join(dir, bundleWorkflowID+".jsonc")); string(definition) != `{"label":"Pulled"}` {
		t.Fatalf("definition: %q", definition)
	}
	root, cfg, err := workspace.Locate(filepath.Join(dir, "functions", "hello"))
	if err != nil || root != filepath.Join(dir, "functions", "hello") || cfg.Target.FunctionID != bundleFunctionID {
		t.Fatalf("functions/hello is not the pulled function workspace: root=%q cfg=%+v err=%v", root, cfg, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "functions", "removed", "index.ts")); !os.IsNotExist(err) {
		t.Fatalf("a file missing from the archive survived the pull: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "functions", "hello", "node_modules", "dep", "a.js")); err != nil {
		t.Fatalf("node_modules must survive a pull: %v", err)
	}
	if cfg, err := workspace.Load(dir); err != nil || cfg.Target.WorkflowID != bundleWorkflowID {
		t.Fatalf("the workflow's own config changed: %+v err=%v", cfg, err)
	}
}

func TestFunctionCreateInAWorkflowMakesAPushableWorkspace(t *testing.T) {
	server := newWorkflowServer(t)
	dir := bundleWorkspace(t, workspace.Target{Kind: "workflow", WorkflowID: bundleWorkflowID}, map[string]string{bundleWorkflowID + ".jsonc": "{}"})

	create, _, err := rootCmd.Find([]string{"function", "create"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	create.SetOut(&stdout)
	t.Cleanup(func() { create.SetOut(nil) })
	if err := create.RunE(create, []string{"functions/hello"}); err != nil {
		t.Fatal(err)
	}
	if body := server.bodies["functions"]; body["workflowId"] != bundleWorkflowID || body["slug"] != "hello" {
		t.Fatalf("create body: %v", body)
	}
	if cfg, err := workspace.Load(filepath.Join(dir, "functions", "hello")); err != nil || cfg.Target.FunctionID != bundleFunctionID {
		t.Fatalf("function workspace config: %+v err=%v", cfg, err)
	}

	if _, stderr, err := runTopLevel(t, "push", map[string]string{}); err != nil {
		t.Fatalf("push of the workflow with the new function: %v stderr=%q", err, stderr)
	}
	if len(server.uploads["function"]) == 0 {
		t.Fatal("the new function was not pushed with its workflow")
	}

	if err := create.RunE(create, []string{"functions/Not_A_Slug"}); err == nil || !strings.Contains(err.Error(), "must start with a lowercase letter") {
		t.Fatalf("an invalid slug was accepted: %v", err)
	}
}
