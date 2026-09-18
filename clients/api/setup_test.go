package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestAppEnvSetupForwardsOnlyExplicitThreadHint(t *testing.T) {
	previous := testTokenOverride
	testTokenOverride = ""
	t.Cleanup(func() { testTokenOverride = previous })
	for _, threadID := range []string{"", "36b02dd1-2533-4421-82f9-3251a7bc2a23"} {
		t.Run(threadID, func(t *testing.T) {
			t.Setenv("MAJOR_TOKEN", "sandbox-test-token")
			t.Setenv("MAJOR_CHAT_THREAD_ID", threadID)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if r.Header.Get("Authorization") != "Bearer sandbox-test-token" {
					t.Error("must retain sandbox authentication")
				}
				if threadID == "" {
					if _, exists := body["chatThreadId"]; exists {
						t.Error("external calls must omit attribution")
					}
				} else if body["chatThreadId"] != threadID {
					t.Errorf("thread hint = %v", body["chatThreadId"])
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"delivery":"thread","message":"Prompted"}`)
			}))
			defer server.Close()
			if _, err := NewClient(server.URL).RequestAppEnvSetup("app", AppEnvSetupRequest{Kind: "env", Keys: []string{"API_KEY"}}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
