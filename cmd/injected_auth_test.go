package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/zalando/go-keyring"
)

const sentinelCredential = "sentinel-injected-credential"

func TestLoginRejectsInjectedTokenBeforeRootVersionCheck(t *testing.T) {
	testInjectedAuthManagementCommandPath(t, "user", "login")
}

func TestLogoutRejectsInjectedTokenBeforeRootVersionCheck(t *testing.T) {
	testInjectedAuthManagementCommandPath(t, "user", "logout")
}

func testInjectedAuthManagementCommandPath(t *testing.T, args ...string) {
	t.Helper()

	origVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = origVersion })

	t.Setenv("MAJOR_TOKEN", sentinelCredential)
	keyring.MockInit()
	if err := mjrToken.StoreToken("keychain-token"); err != nil {
		t.Fatal(err)
	}

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
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for externally managed credential")
	}
	if !strings.Contains(err.Error(), "externally managed") {
		t.Fatalf("error = %v, want externally managed", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("HTTP calls = %d, want 0 (version check must not run)", got)
	}
	combined := outBuf.String() + errBuf.String() + err.Error()
	if strings.Contains(combined, sentinelCredential) {
		t.Fatalf("credential leaked in output: stdout=%q stderr=%q err=%v", outBuf.String(), errBuf.String(), err)
	}

	t.Setenv("MAJOR_TOKEN", "")
	got, keyErr := mjrToken.GetToken()
	if keyErr != nil {
		t.Fatalf("keyring token missing after command: %v", keyErr)
	}
	if got != "keychain-token" {
		t.Fatalf("keyring token = %q, want keychain-token", got)
	}
}
