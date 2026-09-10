package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/major-technology/cli/utils"
	"github.com/zalando/go-keyring"
)

func TestRootHelpRegistersNonInteractiveFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("non-interactive")
	if flag == nil {
		t.Fatal("root must inherit --non-interactive")
	}
	if flag.DefValue != "false" {
		t.Fatalf("default = %q, want false (must not imply --yes)", flag.DefValue)
	}
}

func TestLoginNonInteractiveFailsBeforeDeviceFlow(t *testing.T) {
	keyring.MockInit()
	t.Setenv("MAJOR_TOKEN", "")

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		t.Errorf("unexpected HTTP call: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MAJOR_API_URL", srv.URL+"/cli")

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetIn(strings.NewReader(""))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetIn(nil)
		rootCmd.SetArgs(nil)
	})

	rootCmd.SetArgs([]string{"--non-interactive", "user", "login"})
	err := runWithDeadline(t, 8*time.Second, rootCmd.Execute)
	if err == nil {
		t.Fatal("non-interactive login must fail")
	}
	msg := err.Error() + outBuf.String() + errBuf.String()
	if !strings.Contains(msg, "MAJOR_TOKEN") {
		t.Fatalf("error must name MAJOR_TOKEN, got %v stdout=%q stderr=%q", err, outBuf.String(), errBuf.String())
	}
	if !strings.Contains(strings.ToLower(msg), "login") {
		t.Fatalf("error must mention prior human login, got %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("HTTP calls = %d, want 0 (must not start device login)", got)
	}
}

func TestDocsNonInteractivePrintsURLWithoutOpening(t *testing.T) {
	orig := utils.BrowserStart
	opened := []string{}
	utils.BrowserStart = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	t.Cleanup(func() { utils.BrowserStart = orig })

	outBuf := &bytes.Buffer{}
	docsCmd.SetOut(outBuf)
	docsCmd.SetErr(outBuf)
	docsCmd.SetIn(strings.NewReader(""))
	t.Cleanup(func() {
		docsCmd.SetOut(nil)
		docsCmd.SetErr(nil)
		docsCmd.SetIn(nil)
	})

	err := runWithDeadline(t, 8*time.Second, func() error {
		return docsCmd.RunE(docsCmd, nil)
	})
	if err != nil {
		t.Fatalf("docs in non-interactive mode: %v", err)
	}
	if len(opened) != 0 {
		t.Fatalf("opened browser: %v", opened)
	}
	if !strings.Contains(outBuf.String(), "https://docs.major.build/") {
		t.Fatalf("output must print destination, got %q", outBuf.String())
	}
}

func TestCreateHelpShowsCompleteNonInteractiveInvocation(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	rootCmd.SetArgs([]string{"app", "create", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, want := range []string{"--name", "--description", "--theme-id", "--github-user"} {
		if !strings.Contains(help, want) {
			t.Errorf("app create help missing %q\n%s", want, help)
		}
	}
}

func runWithDeadline(t *testing.T, d time.Duration, fn func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		t.Fatalf("command did not complete within %s (possible prompt)", d)
		return nil
	}
}
