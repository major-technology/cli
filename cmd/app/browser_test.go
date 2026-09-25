package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/workspace"
)

type recordedToolCall struct {
	name string
	args map[string]any
}

func fakeSandbox(t *testing.T, reply string) *[]recordedToolCall {
	t.Helper()
	calls := &[]recordedToolCall{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var args map[string]any
		if err := json.Unmarshal(body, &args); err != nil {
			t.Errorf("body is not JSON: %s", body)
		}
		*calls = append(*calls, recordedToolCall{name: strings.TrimPrefix(r.URL.Path, "/tools/"), args: args})
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, reply)
	}))
	t.Cleanup(server.Close)

	t.Setenv(sandboxToolsURLEnv, server.URL+"/tools/")

	return calls
}

func runBrowser(t *testing.T, args ...string) (string, error) {
	t.Helper()
	sub, rest, err := browserCmd.Find(args)
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.ParseFlags(rest); err != nil {
		t.Fatal(err)
	}
	if err := sub.ValidateArgs(sub.Flags().Args()); err != nil {
		return "", err
	}

	var out bytes.Buffer
	sub.SetOut(&out)
	sub.SetErr(&out)
	err = sub.RunE(sub, sub.Flags().Args())
	return out.String(), err
}

func TestBrowserSubcommandsRegistered(t *testing.T) {
	want := map[string]bool{"navigate": false, "snapshot": false, "screenshot": false, "console": false, "network": false, "wait-for": false, "close": false}

	for _, sub := range browserCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("app browser missing subcommand %q", name)
		}
	}
}

func TestBrowserNavigateForwardsURLAndPrintsText(t *testing.T) {
	calls := fakeSandbox(t, `{"content":[{"type":"text","text":"### Page\n- Page URL: http://localhost:3000/dashboard"}]}`)

	out, err := runBrowser(t, "navigate", "/dashboard")
	if err != nil {
		t.Fatalf("navigate: %v", err)
	}

	if len(*calls) != 1 || (*calls)[0].name != "browser_navigate" || (*calls)[0].args["url"] != "/dashboard" {
		t.Fatalf("unexpected calls: %+v", *calls)
	}
	if !strings.Contains(out, "Page URL: http://localhost:3000/dashboard") {
		t.Fatalf("output missing tool text: %q", out)
	}
}

func TestBrowserConsoleSendsLevel(t *testing.T) {
	calls := fakeSandbox(t, `{"content":[{"type":"text","text":"none"}]}`)

	if _, err := runBrowser(t, "console", "--level", "warning"); err != nil {
		t.Fatalf("console: %v", err)
	}

	if (*calls)[0].name != "browser_console_messages" || (*calls)[0].args["level"] != "warning" {
		t.Fatalf("unexpected calls: %+v", *calls)
	}
}

func TestBrowserWaitForRequiresACondition(t *testing.T) {
	fakeSandbox(t, `{"content":[]}`)

	if _, err := runBrowser(t, "wait-for"); err == nil {
		t.Fatal("wait-for with no flag must fail")
	}
}

func TestBrowserToolErrorBecomesCommandError(t *testing.T) {
	fakeSandbox(t, `{"content":[{"type":"text","text":"Can't navigate to url https://example.com."}],"isError":true}`)

	_, err := runBrowser(t, "navigate", "https://example.com")
	if err == nil || !strings.Contains(err.Error(), "Can't navigate") {
		t.Fatalf("want the tool error, got %v", err)
	}
}

func TestBrowserScreenshotSavesImageUnderSessionFiles(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47}
	calls := fakeSandbox(t, fmt.Sprintf(`{"content":[{"type":"text","text":"saved"},{"type":"image","data":%q,"mimeType":"image/png"}]}`, base64.StdEncoding.EncodeToString(png)))

	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: testOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: testAppID},
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	out, err := runBrowser(t, "screenshot", "../../dash.png", "--full-page")
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}

	if (*calls)[0].name != "browser_take_screenshot" || (*calls)[0].args["fullPage"] != true {
		t.Fatalf("unexpected calls: %+v", *calls)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".session-files", "dash.png"))
	if err != nil || !bytes.Equal(got, png) {
		t.Fatalf("screenshot not saved: %v", err)
	}
	if !strings.Contains(out, filepath.Join(".session-files", "dash.png")) {
		t.Fatalf("output missing path: %q", out)
	}
}

func TestBrowserOutsideSandboxExplains(t *testing.T) {
	t.Setenv(sandboxToolsURLEnv, "")

	_, err := runBrowser(t, "snapshot")
	if err == nil || !strings.Contains(err.Error(), "only inside a Major sandbox") {
		t.Fatalf("want the sandbox-only error, got %v", err)
	}
}
