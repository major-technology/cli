package app

import (
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

func TestThemeSubcommandsRegistered(t *testing.T) {
	want := map[string]bool{"list": false, "get": false, "apply": false}

	for _, sub := range themeCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("app theme missing subcommand %q", name)
		}
	}
}

func TestThemeRegisteredOnAppCmd(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Name() == "theme" {
			return
		}
	}

	t.Fatal("app theme not registered on the app command")
}

func TestThemeApplyRequiresThemeID(t *testing.T) {
	if err := themeApplyCmd.Args(themeApplyCmd, []string{}); err == nil {
		t.Fatal("app theme apply must require a theme id")
	}
}

func TestThemeApplyWritesFilesLocally(t *testing.T) {
	var pinned string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/info"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"test-app","name":"Test","deployStatus":"not_deployed","appUrl":null}`, niAppID, niOrgID)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/theme"):
			body, _ := io.ReadAll(r.Body)
			pinned = string(body)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/theme-files"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"css":":root{--base:#101010}","themeModule":"export const theme = {}","logoComponent":"export function Logo(){return null}"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	dir := newThemeTestRepo(t, srv.URL)

	if err := themeApplyCmd.RunE(themeApplyCmd, []string{"11111111-1111-4111-8111-111111111111"}); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	if !strings.Contains(pinned, "11111111-1111-4111-8111-111111111111") {
		t.Fatalf("theme id was not pinned, got body %q", pinned)
	}

	for _, rel := range []string{"app/theme.css", "lib/theme.ts", "components/ui/logo.tsx"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("apply did not write %s: %v", rel, err)
		}
	}
}

// newThemeTestRepo creates a temp git repo configured as an app workspace pointing at
// baseURL, chdirs into it for the test's duration, and returns its path.
func newThemeTestRepo(t *testing.T, baseURL string) string {
	t.Helper()

	dir := gitRepo(t)

	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: niOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: niAppID},
	}); err != nil {
		t.Fatal(err)
	}
	if err := workspace.IgnoreLocalConfig(dir); err != nil {
		t.Fatal(err)
	}

	restoreAPIClient(t, baseURL)
	t.Chdir(dir)

	return dir
}
