package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/zalando/go-keyring"
)

func TestInjectedTokenSentOnceOn401(t *testing.T) {
	prevOverride := testTokenOverride
	testTokenOverride = ""
	t.Cleanup(func() { testTokenOverride = prevOverride })

	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	keyring.MockInit()
	if err := mjrToken.StoreToken("keychain-token"); err != nil {
		t.Fatal(err)
	}

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer test-injected-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-injected-token")
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Error: &AppErrorDetail{
				InternalCode: ErrorCodeUnauthorized,
				ErrorString:  "unauthorized",
				StatusCode:   http.StatusUnauthorized,
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL)
	_, err := client.VerifyToken()
	if err == nil {
		t.Fatal("expected 401 error")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1 (no alternate-token retry)", got)
	}
}
