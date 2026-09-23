package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

const (
	contractAppID     = "11111111-1111-4111-8111-111111111111"
	contractOrgID     = "22222222-2222-4222-8222-222222222222"
	contractEnvID     = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	contractVersionID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	contractLogCursor = "opaque-cursor-abcdefghijklmnopqrstuvwxyz-0123456789-full"
	contractToken     = "test-injected-token"
	contractInfoBody  = `{"applicationId":"11111111-1111-4111-8111-111111111111","organizationId":"22222222-2222-4222-8222-222222222222","urlSlug":"prototype","name":"Prototype","deployStatus":"not_deployed","appUrl":null}`
)

func TestRootDoesNotExposeJSONFlag(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("json") != nil {
		t.Fatal("root --json must not exist; JSON is per supported command")
	}
	if rootCmd.Flags().Lookup("json") != nil {
		t.Fatal("root --json must not exist; JSON is per supported command")
	}
}

func TestSupportedCommandsExposeJSONFlags(t *testing.T) {
	paths := [][]string{
		{"app", "info"},
		{"app", "logs"},
		{"app", "deploy-status"},
		{"vars", "list"},
		{"vars", "get"},
		{"vars", "set"},
		{"vars", "unset"},
		{"resource", "list"},
		{"resource", "add"},
		{"resource", "remove"},
	}
	for _, path := range paths {
		cmd, _, err := rootCmd.Find(path)
		if err != nil {
			t.Fatalf("Find %v: %v", path, err)
		}
		if cmd.Flags().Lookup("json") == nil {
			t.Errorf("%s missing --json", strings.Join(path, " "))
		}
	}
}

func TestInfoJSONWritesExistingShapeAndFullIDs(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
	}), []string{"app", "info"}, nil, true, nil)
	if err != nil {
		t.Fatalf("app info --json: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["applicationId"] != contractAppID {
		t.Fatalf("applicationId = %#v", result["applicationId"])
	}
	if result["name"] != "Prototype" || result["deployStatus"] != "not_deployed" {
		t.Fatalf("result = %#v", result)
	}
	if _, ok := result["appUrl"]; !ok {
		t.Fatal("existing info JSON must keep appUrl")
	}
	if stderr != "" && strings.Contains(stderr, contractToken) {
		t.Fatalf("token leaked on stderr: %q", stderr)
	}
}

func TestInfoAPIErrorDoesNotPrintAppIDOrSucceed(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":{"internal_code":9999,"error_string":"info unavailable","status_code":500}}`)
		},
	}), []string{"app", "info"}, nil, true, nil)
	if err == nil {
		t.Fatalf("auth/network info failure must not succeed, stdout=%q", stdout)
	}
	if strings.Contains(stdout, contractAppID) || strings.Contains(stdout, "Application ID:") {
		t.Fatalf("must not print app ID on info failure, stdout=%q", stdout)
	}
	assertNoSuccessJSON(t, stdout)
	assertNoTokenLeak(t, stdout, stderr, err)
}

func TestLogsJSONKeepsFullCursorAndHintsStderr(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /applications/" + contractAppID + "/logs": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("nextToken") != "" {
				t.Errorf("first page must not send nextToken, got %q", r.URL.RawQuery)
			}
			writeJSON(w, `{"logs":[{"ts":"2026-09-10T00:00:00Z","log":"hello"}],"nextToken":"`+contractLogCursor+`"}`)
		},
	}), []string{"app", "logs"}, nil, true, nil)
	if err != nil {
		t.Fatalf("app logs --json: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["nextToken"] != contractLogCursor {
		t.Fatalf("nextToken truncated or missing: %#v", result["nextToken"])
	}
	if !strings.Contains(stderr, contractLogCursor) {
		t.Fatalf("pagination hint must go to stderr with full cursor, stderr=%q", stderr)
	}
	if strings.Contains(stdout, "more logs") {
		t.Fatalf("hint leaked onto stdout: %q", stdout)
	}
}

func TestLogsJSONEmptyResults(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /applications/" + contractAppID + "/logs": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"logs":[]}`)
		},
	}), []string{"app", "logs"}, nil, true, nil)
	if err != nil {
		t.Fatalf("app logs --json empty: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	logs, ok := result["logs"].([]any)
	if !ok || len(logs) != 0 {
		t.Fatalf("logs = %#v", result["logs"])
	}
}

func TestDeployStatusJSONExistingShape(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"POST /applications/versions/status": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"status":"deployed","app_url":"https://prototype.example.test"}`)
		},
	}), []string{"app", "deploy-status"}, nil, true, map[string]string{"version-id": contractVersionID})
	if err != nil {
		t.Fatalf("deploy-status --json: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["status"] != "deployed" {
		t.Fatalf("status = %#v", result["status"])
	}
	if result["appUrl"] != "https://prototype.example.test" {
		t.Fatalf("existing deploy-status JSON must keep appUrl, got %#v", result["appUrl"])
	}
}

func TestVarsListJSONEmptyAndGetKeepsValues(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /application/" + contractAppID + "/environment": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"environmentId":"`+contractEnvID+`","environmentName":"development"}`)
		},
		"GET /application/" + contractAppID + "/env-variables": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"envVariables":[]}`)
		},
	}
	stdout, stderr, err := runContractCommand(t, contractServer(t, routes), []string{"vars", "list"}, nil, true, nil)
	if err != nil {
		t.Fatalf("vars list --json: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["environment"] != "development" {
		t.Fatalf("environment = %#v", result["environment"])
	}
	vars, ok := result["variables"].([]any)
	if !ok || len(vars) != 0 {
		t.Fatalf("variables = %#v", result["variables"])
	}

	routes["GET /application/"+contractAppID+"/env-variables"] = func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"envVariables":[{"key":"CLI_PROTOTYPE","values":[{"environmentId":"`+contractEnvID+`","value":"secret-value"}]}]}`)
	}
	stdout, stderr, err = runContractCommand(t, contractServer(t, routes), []string{"vars", "get"}, []string{"CLI_PROTOTYPE"}, true, nil)
	if err != nil {
		t.Fatalf("vars get --json: %v stderr=%q", err, stderr)
	}
	got := decodeOneObject(t, stdout)
	if got["key"] != "CLI_PROTOTYPE" || got["value"] != "secret-value" || got["environment"] != "development" {
		t.Fatalf("get JSON changed shape: %#v", got)
	}
}

func TestVarsSetJSONOmitsValue(t *testing.T) {
	var setBody string
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /application/" + contractAppID + "/environment": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"environmentId":"`+contractEnvID+`","environmentName":"development"}`)
		},
		"POST /application/" + contractAppID + "/env-variables/set": func(w http.ResponseWriter, r *http.Request) {
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(r.Body)
			setBody = buf.String()
			writeJSON(w, `{"id":"var-1","key":"CLI_PROTOTYPE","created":true}`)
		},
	}), []string{"vars", "set"}, []string{"CLI_PROTOTYPE=secret-value"}, true, nil)
	if err != nil {
		t.Fatalf("vars set --json: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["key"] != "CLI_PROTOTYPE" || result["environment"] != "development" || result["updated"] != true {
		t.Fatalf("set JSON = %#v", result)
	}
	if _, exists := result["value"]; exists {
		t.Fatalf("mutation JSON must not include value: %#v", result)
	}
	if strings.Contains(stdout, "secret-value") {
		t.Fatalf("value leaked on stdout: %q", stdout)
	}
	if setBody == "" {
		t.Fatal("expected set API call")
	}
}

func TestVarsUnsetJSONOmitsValue(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /application/" + contractAppID + "/environment": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"environmentId":"`+contractEnvID+`","environmentName":"development"}`)
		},
		"DELETE /application/" + contractAppID + "/env-variables/by-key/CLI_PROTOTYPE": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"deleted":true,"removedRow":true}`)
		},
	}), []string{"vars", "unset"}, []string{"CLI_PROTOTYPE"}, true, map[string]string{"yes": "true"})
	if err != nil {
		t.Fatalf("vars unset --json: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["key"] != "CLI_PROTOTYPE" || result["environment"] != "development" || result["deleted"] != true {
		t.Fatalf("unset JSON = %#v", result)
	}
	if _, exists := result["value"]; exists {
		t.Fatalf("mutation JSON must not include value: %#v", result)
	}
	if _, exists := result["allEnvironments"]; exists {
		t.Fatalf("single-environment JSON must omit allEnvironments: %#v", result)
	}
}

func TestVarsUnsetJSONAllEnvironmentsOmitsEnvironmentName(t *testing.T) {
	var deleteQuery string
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"DELETE /application/" + contractAppID + "/env-variables/by-key/CLI_PROTOTYPE": func(w http.ResponseWriter, r *http.Request) {
			deleteQuery = r.URL.RawQuery
			writeJSON(w, `{"deleted":true,"removedRow":true}`)
		},
	}), []string{"vars", "unset"}, []string{"CLI_PROTOTYPE"}, true, map[string]string{"yes": "true", "all-environments": "true"})
	if err != nil {
		t.Fatalf("vars unset --all-environments --json: %v stderr=%q", err, stderr)
	}
	if !strings.Contains(deleteQuery, "allEnvironments=true") {
		t.Fatalf("expected allEnvironments=true query, got %q", deleteQuery)
	}
	if strings.Contains(deleteQuery, "environmentId=") {
		t.Fatalf("all-environments delete must not send environmentId, query=%q", deleteQuery)
	}
	result := decodeOneObject(t, stdout)
	if result["key"] != "CLI_PROTOTYPE" || result["allEnvironments"] != true || result["deleted"] != true {
		t.Fatalf("all-environments unset JSON = %#v", result)
	}
	if _, exists := result["environment"]; exists {
		t.Fatalf("all-environments JSON must omit environment: %#v", result)
	}
	if _, exists := result["value"]; exists {
		t.Fatalf("mutation JSON must not include value: %#v", result)
	}
}

func TestVarsUnsetJSONSingleEnvironmentNamedAll(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /application/" + contractAppID + "/environment": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"environmentId":"`+contractEnvID+`","environmentName":"all"}`)
		},
		"DELETE /application/" + contractAppID + "/env-variables/by-key/CLI_PROTOTYPE": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("environmentId"); got != contractEnvID {
				t.Errorf("environmentId = %q", got)
			}
			if r.URL.Query().Get("allEnvironments") != "" {
				t.Errorf("single-env delete must not send allEnvironments, query=%q", r.URL.RawQuery)
			}
			writeJSON(w, `{"deleted":true,"removedRow":true}`)
		},
	}), []string{"vars", "unset"}, []string{"CLI_PROTOTYPE"}, true, map[string]string{"yes": "true"})
	if err != nil {
		t.Fatalf("vars unset --json env named all: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["key"] != "CLI_PROTOTYPE" || result["environment"] != "all" || result["deleted"] != true {
		t.Fatalf("unset JSON for env named all = %#v", result)
	}
	if _, exists := result["allEnvironments"]; exists {
		t.Fatalf("named environment all must not set allEnvironments: %#v", result)
	}
}

func TestResourceListJSONKeepsFullIDs(t *testing.T) {
	const resourceID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /verify": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"active":true,"user_id":"user-1"}`)
		},
		"POST /resources": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"resources":[{"id":"`+resourceID+`","name":"db","type":"postgres","description":"db"}]}`)
		},
		"GET /applications/" + contractAppID + "/resources": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"resources":[{"id":"`+resourceID+`","name":"db","type":"postgres","description":"db"}]}`)
		},
	}), []string{"resource", "list"}, nil, true, nil)
	if err != nil {
		t.Fatalf("resource list --json: %v stderr=%q", err, stderr)
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(stdout)))
	var result []any
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("resource list JSON: %v stdout=%q", err, stdout)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("unexpected trailing stdout: %v", err)
	}
	if bytes.Contains([]byte(stdout), []byte{0x1b}) {
		t.Fatalf("ANSI in stdout: %q", stdout)
	}
	if len(result) != 1 {
		t.Fatalf("resources = %#v", result)
	}
	row, _ := result[0].(map[string]any)
	if row["id"] != resourceID {
		t.Fatalf("id truncated or missing: %#v", row["id"])
	}
}

func TestJSONAPIErrorsHaveNoSuccessResult(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{name: "401", status: http.StatusUnauthorized, body: `{"error":{"internal_code":2000,"error_string":"unauthorized","status_code":401}}`},
		{name: "403", status: http.StatusForbidden, body: `{"error":{"internal_code":4001,"error_string":"forbidden","status_code":403}}`},
		{name: "500", status: http.StatusInternalServerError, body: `{"error":{"internal_code":9999,"error_string":"internal","status_code":500}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
				"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.body)
				},
			}), []string{"app", "info"}, nil, true, nil)
			if err == nil {
				t.Fatalf("expected API %s to fail, stdout=%q", tc.name, stdout)
			}
			assertNoSuccessJSON(t, stdout)
			assertNoTokenLeak(t, stdout, stderr, err)
		})
	}
}

func TestOptionalUpgradeWarningGoesToStderrNotJSON(t *testing.T) {
	orig := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = orig })

	writeAppWorkspace(t)
	t.Setenv("MAJOR_TOKEN", contractToken)
	restoreAPIClient(t, contractServer(t, map[string]http.HandlerFunc{
		"POST /version/check": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"canUpgrade":true,"latestVersion":"1.2.4"}`)
		},
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /application/" + contractAppID + "/environment": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"environmentId":"`+contractEnvID+`","environmentName":"development"}`)
		},
		"GET /application/" + contractAppID + "/env-variables": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"envVariables":[]}`)
		},
	}))

	cmd, _, err := rootCmd.Find([]string{"vars", "list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("json", "false") })

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(""))
	t.Cleanup(func() {
		cmd.SetOut(nil)
		cmd.SetErr(nil)
		cmd.SetIn(nil)
	})

	if err := rootPersistentPreRunE(cmd, nil); err != nil {
		t.Fatalf("root pre-run: %v", err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("vars list --json: %v stderr=%q", err, stderr.String())
	}

	result := decodeOneObject(t, stdout.String())
	if result["environment"] != "development" {
		t.Fatalf("JSON corrupted by warning: %#v stdout=%q", result, stdout.String())
	}
	if !strings.Contains(stderr.String(), "new version") && !strings.Contains(stderr.String(), "major update") {
		t.Fatalf("upgrade warning must be on stderr, stderr=%q stdout=%q", stderr.String(), stdout.String())
	}
	if strings.Contains(stdout.String(), "new version") || strings.Contains(stdout.String(), "major update") {
		t.Fatalf("upgrade warning leaked onto stdout: %q", stdout.String())
	}
}

func TestCheckVersionWarningUsesStderr(t *testing.T) {
	origClient := singletons.GetAPIClient()
	t.Cleanup(func() { singletons.SetAPIClient(origClient) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/version/check" {
			writeJSON(w, `{"canUpgrade":true}`)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	singletons.SetAPIClient(api.NewClient(srv.URL))

	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := middleware.CheckVersion("1.2.3")(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("version warning on stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "major update") {
		t.Fatalf("version warning missing on stderr: %q", stderr.String())
	}
}

func TestPrintErrorWritesToStderrWithoutToken(t *testing.T) {
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	clierrors.PrintError(cmd, clierrors.ErrorUnauthorized, false)
	if stdout.Len() != 0 {
		t.Fatalf("PrintError wrote stdout: %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("PrintError must write stderr")
	}
	if strings.Contains(stderr.String(), "Authorization") || strings.Contains(stderr.String(), "Bearer") || strings.Contains(stderr.String(), contractToken) {
		t.Fatalf("headers or token leaked: %q", stderr.String())
	}
}

func TestInjectedTokenErrorIsPlainText(t *testing.T) {
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	clierrors.PrintError(cmd, &clierrors.CLIError{Title: "Use the MCP run_agent tool instead."}, true)
	if got := stderr.String(); got != "Error: Use the MCP run_agent tool instead.\n" {
		t.Fatalf("plain error = %q", got)
	}
	if stdout.Len() != 0 {
		t.Fatalf("error wrote stdout: %q", stdout.String())
	}
}

func TestNonTTYJSONStillWorks(t *testing.T) {
	stdout, stderr, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
	}), []string{"app", "info"}, nil, true, nil)
	if err != nil {
		t.Fatalf("non-TTY json: %v stderr=%q", err, stderr)
	}
	_ = decodeOneObject(t, stdout)
}

func TestVarsListHumanProseStaysOnStdout(t *testing.T) {
	stdout, _, err := runContractCommand(t, contractServer(t, map[string]http.HandlerFunc{
		"GET /applications/" + contractAppID + "/info": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, contractInfoBody)
		},
		"GET /application/" + contractAppID + "/environment": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"environmentId":"`+contractEnvID+`","environmentName":"development"}`)
		},
		"GET /application/" + contractAppID + "/env-variables": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"envVariables":[]}`)
		},
	}), []string{"vars", "list"}, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "No variables set.") {
		t.Fatalf("human output missing, stdout=%q", stdout)
	}
}

func runContractCommand(t *testing.T, baseURL string, path []string, args []string, jsonOut bool, flags map[string]string) (string, string, error) {
	t.Helper()
	writeAppWorkspace(t)
	t.Setenv("MAJOR_TOKEN", contractToken)
	restoreAPIClient(t, baseURL)

	cmd, _, err := rootCmd.Find(path)
	if err != nil {
		t.Fatalf("Find %v: %v", path, err)
	}
	if f := cmd.Flags().Lookup("json"); f != nil {
		value := "false"
		if jsonOut {
			value = "true"
		}
		if err := cmd.Flags().Set("json", value); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Flags().Set("json", "false") })
	} else if jsonOut {
		return "", "", fmt.Errorf("%s missing --json", strings.Join(path, " "))
	}
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set flag %s: %v", name, err)
		}
		t.Cleanup(func() { _ = cmd.Flags().Set(name, cmd.Flags().Lookup(name).DefValue) })
	}

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(""))
	t.Cleanup(func() {
		cmd.SetOut(nil)
		cmd.SetErr(nil)
		cmd.SetIn(nil)
	})

	runErr := cmd.RunE(cmd, args)
	return stdout.String(), stderr.String(), runErr
}

func contractServer(t *testing.T, routes map[string]http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if handler, ok := routes[key]; ok {
			handler(w, r)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, body)
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

func assertNoSuccessJSON(t *testing.T, stdout string) {
	t.Helper()
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	var result map[string]any
	if err := decoder.Decode(&result); err != nil {
		return
	}
	if _, hasID := result["applicationId"]; hasID {
		t.Fatalf("success-shaped JSON on error: %#v", result)
	}
	if updated, ok := result["updated"].(bool); ok && updated {
		t.Fatalf("success-shaped JSON on error: %#v", result)
	}
	if deleted, ok := result["deleted"].(bool); ok && deleted {
		t.Fatalf("success-shaped JSON on error: %#v", result)
	}
	if attached, ok := result["attached"].(bool); ok && attached {
		t.Fatalf("success-shaped JSON on error: %#v", result)
	}
}

func assertNoTokenLeak(t *testing.T, stdout, stderr string, err error) {
	t.Helper()
	combined := stdout + stderr
	if err != nil {
		combined += err.Error()
	}
	if strings.Contains(combined, contractToken) || strings.Contains(combined, "Authorization") || strings.Contains(combined, "Bearer") {
		t.Fatalf("credential or header leaked: stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
}
