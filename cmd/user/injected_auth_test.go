package user

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	apiClient "github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func newOutputCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}

const sentinelCredential = "sentinel-injected-credential"

func captureProcessAndCommandOutput(t *testing.T, cmd *cobra.Command, fn func() error) (stdout, stderr string, execErr error) {
	t.Helper()
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w
	execErr = fn()
	_ = w.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr
	processOut, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatal(readErr)
	}
	stdout = outBuf.String() + string(processOut)
	stderr = errBuf.String()
	return stdout, stderr, execErr
}

func setupInjectedAuthCommand(t *testing.T) *atomic.Int32 {
	t.Helper()
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
	singletons.SetAPIClient(apiClient.NewClient(srv.URL))
	return &requests
}

func assertNoCredentialExposure(t *testing.T, stdout, stderr string, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error for externally managed credential")
	}
	if !strings.Contains(err.Error(), "externally managed") {
		t.Fatalf("error = %v, want externally managed", err)
	}
	combined := stdout + stderr + err.Error()
	if strings.Contains(combined, sentinelCredential) {
		t.Fatalf("credential leaked in output: stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
}

func assertKeyringUnchanged(t *testing.T) {
	t.Helper()
	t.Setenv("MAJOR_TOKEN", "")
	got, err := mjrToken.GetToken()
	if err != nil {
		t.Fatalf("keyring token missing after command: %v", err)
	}
	if got != "keychain-token" {
		t.Fatalf("keyring token = %q, want keychain-token (command must not overwrite or revoke)", got)
	}
}

func TestLoginRejectsInjectedToken(t *testing.T) {
	requests := setupInjectedAuthCommand(t)
	cmd := newOutputCommand()
	stdout, stderr, err := captureProcessAndCommandOutput(t, cmd, func() error {
		return runLogin(cmd)
	})
	assertNoCredentialExposure(t, stdout, stderr, err)
	if requests.Load() != 0 {
		t.Fatalf("HTTP calls = %d, want 0", requests.Load())
	}
	assertKeyringUnchanged(t)
}

func TestLogoutRejectsInjectedToken(t *testing.T) {
	requests := setupInjectedAuthCommand(t)
	cmd := newOutputCommand()
	stdout, stderr, err := captureProcessAndCommandOutput(t, cmd, func() error {
		return runLogout(cmd)
	})
	assertNoCredentialExposure(t, stdout, stderr, err)
	if requests.Load() != 0 {
		t.Fatalf("HTTP calls = %d, want 0", requests.Load())
	}
	assertKeyringUnchanged(t)
}

func TestTokenDisplayRejectsInjectedToken(t *testing.T) {
	requests := setupInjectedAuthCommand(t)
	cmd := newOutputCommand()
	stdout, stderr, err := captureProcessAndCommandOutput(t, cmd, func() error {
		return runToken()
	})
	assertNoCredentialExposure(t, stdout, stderr, err)
	if requests.Load() != 0 {
		t.Fatalf("HTTP calls = %d, want 0", requests.Load())
	}
	if strings.Contains(stdout, sentinelCredential) || strings.Contains(stdout, "keychain-token") {
		t.Fatalf("token display emitted a token: %q", stdout)
	}
}
