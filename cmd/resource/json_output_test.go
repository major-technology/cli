package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

const (
	jsonResourceID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
)

func TestResourceCommandsExposeJSONFlag(t *testing.T) {
	for _, cmd := range []*cobra.Command{listCmd, addCmd, removeCmd} {
		if cmd.Flags().Lookup("json") == nil {
			t.Errorf("%s missing --json", cmd.Name())
		}
	}
}

func TestRunAddJSONAfterAPIAndLocalGeneration(t *testing.T) {
	dir := preparedResourceWorkspace(t)
	var saved atomic.Int32
	restoreAPIClient(t, resourceMutationServer(t, &saved, true))

	orig := addResourcesToProject
	var generated atomic.Int32
	addResourcesToProject = func(cmd *cobra.Command, projectDir string, resources []api.ResourceItem, applicationID string) error {
		generated.Add(1)
		if applicationID != niAppID {
			t.Errorf("applicationID = %q", applicationID)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "child chatter from generator")
		return nil
	}
	t.Cleanup(func() { addResourcesToProject = orig })

	flagAddJSON = true
	flagAddResourceID = jsonResourceID
	t.Cleanup(func() {
		flagAddJSON = false
		flagAddResourceID = ""
	})

	cmd, stdout, stderr := jsonCommand(t)
	if err := runAdd(cmd); err != nil {
		t.Fatalf("runAdd: %v stderr=%q", err, stderr.String())
	}
	if saved.Load() == 0 {
		t.Fatal("expected SaveApplicationResources")
	}
	if generated.Load() != 1 {
		t.Fatalf("local generation calls = %d, want 1", generated.Load())
	}
	result := decodeOneObject(t, stdout.String())
	if result["resourceId"] != jsonResourceID || result["attached"] != true {
		t.Fatalf("add JSON = %#v", result)
	}
	if strings.Contains(stdout.String(), "child chatter") {
		t.Fatalf("child chatter leaked onto stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "child chatter") {
		t.Fatalf("child chatter must go to stderr for JSON, stderr=%q", stderr.String())
	}
	_ = dir
}

func TestRunAddJSONNotEmittedWhenGenerationFails(t *testing.T) {
	preparedResourceWorkspace(t)
	var saved atomic.Int32
	restoreAPIClient(t, resourceMutationServer(t, &saved, true))

	orig := addResourcesToProject
	addResourcesToProject = func(cmd *cobra.Command, projectDir string, resources []api.ResourceItem, applicationID string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "partial chatter")
		return fmt.Errorf("generation failed")
	}
	t.Cleanup(func() { addResourcesToProject = orig })

	flagAddJSON = true
	flagAddResourceID = jsonResourceID
	t.Cleanup(func() {
		flagAddJSON = false
		flagAddResourceID = ""
	})

	cmd, stdout, _ := jsonCommand(t)
	if err := runAdd(cmd); err == nil {
		t.Fatal("expected generation failure")
	}
	if saved.Load() == 0 {
		t.Fatal("API mutation still happens before generation")
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("must not emit JSON after generation failure, stdout=%q", stdout.String())
	}
}

func TestRunRemoveJSONAfterAPIAndLocalGeneration(t *testing.T) {
	dir := preparedResourceWorkspace(t)
	if err := writeLocalResource(dir, jsonResourceID); err != nil {
		t.Fatal(err)
	}
	var saved atomic.Int32
	restoreAPIClient(t, resourceMutationServer(t, &saved, false))

	orig := addResourcesToProject
	addResourcesToProject = func(cmd *cobra.Command, projectDir string, resources []api.ResourceItem, applicationID string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "remove chatter")
		return nil
	}
	t.Cleanup(func() { addResourcesToProject = orig })

	flagRemoveJSON = true
	flagRemoveResourceID = jsonResourceID
	t.Cleanup(func() {
		flagRemoveJSON = false
		flagRemoveResourceID = ""
	})

	cmd, stdout, stderr := jsonCommand(t)
	if err := runRemove(cmd); err != nil {
		t.Fatalf("runRemove: %v stderr=%q", err, stderr.String())
	}
	if saved.Load() == 0 {
		t.Fatal("expected SaveApplicationResources")
	}
	result := decodeOneObject(t, stdout.String())
	if result["resourceId"] != jsonResourceID || result["attached"] != false {
		t.Fatalf("remove JSON = %#v", result)
	}
	if strings.Contains(stdout.String(), "remove chatter") {
		t.Fatalf("child chatter leaked onto stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "remove chatter") {
		t.Fatalf("child chatter must go to stderr for JSON, stderr=%q", stderr.String())
	}
}

func preparedResourceWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: niOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: niAppID},
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	return dir
}

func restoreAPIClient(t *testing.T, baseURL string) {
	t.Helper()
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(baseURL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })
}

func resourceMutationServer(t *testing.T, saved *atomic.Int32, includeTarget bool) string {
	t.Helper()
	resources := `{"resources":[{"id":"` + jsonResourceID + `","name":"db","type":"postgres","description":"db"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/applications/"+niAppID+"/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID)
		case r.URL.Path == "/resources":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, resources)
		case r.URL.Path == "/application-resources":
			saved.Add(1)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"success":true}`)
		case r.URL.Path == "/applications/"+niAppID+"/resources":
			w.Header().Set("Content-Type", "application/json")
			if includeTarget {
				fmt.Fprint(w, resources)
			} else {
				fmt.Fprint(w, `{"resources":[]}`)
			}
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func jsonCommand(t *testing.T) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "")
	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	child.SetOut(stdout)
	child.SetErr(stderr)
	child.SetIn(strings.NewReader(""))
	return child, stdout, stderr
}

func writeLocalResource(dir, id string) error {
	body := `[{"id":"` + id + `","name":"db","type":"postgres","description":"db","applicationId":"` + niAppID + `"}]`
	return os.WriteFile(dir+"/resources.json", []byte(body), 0644)
}

func decodeOneObject(t *testing.T, stdout string) map[string]any {
	t.Helper()
	if bytes.Contains([]byte(stdout), []byte{0x1b}) {
		t.Fatalf("ANSI in stdout: %q", stdout)
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(stdout)))
	var result map[string]any
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("stdout JSON: %v stdout=%q", err, stdout)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("unexpected trailing stdout: %v extra=%v stdout=%q", err, extra, stdout)
	}
	return result
}
