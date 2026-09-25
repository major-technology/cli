package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestSandboxUploadURLSendsPathAndPrintsURL(t *testing.T) {
	const uploadURL = "https://api.example.test/cli/sandbox-uploads/ticket"
	routes := map[string]http.HandlerFunc{
		"POST /sandbox-uploads": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["destPath"] != "public/logo.png" {
				t.Errorf("body = %v, err = %v", body, err)
			}
			writeJSON(w, `{"uploadUrl":"`+uploadURL+`","expiresAt":"2030-01-01T00:00:00.000Z"}`)
		},
	}

	stdout, stderr, err := runContractCommand(t, contractServer(t, routes), []string{"sandbox", "upload-url"}, nil, true, map[string]string{"path": "public/logo.png"})
	if err != nil {
		t.Fatalf("sandbox upload-url --json: %v stderr=%q", err, stderr)
	}
	result := decodeOneObject(t, stdout)
	if result["uploadUrl"] != uploadURL || result["expiresAt"] != "2030-01-01T00:00:00.000Z" {
		t.Fatalf("result = %#v", result)
	}

	stdout, _, err = runContractCommand(t, contractServer(t, routes), []string{"sandbox", "upload-url"}, nil, false, map[string]string{"path": "public/logo.png"})
	if err != nil {
		t.Fatalf("sandbox upload-url: %v", err)
	}
	if !strings.Contains(stdout, "curl -T <file> '"+uploadURL+"'") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestSandboxUploadURLOutsideSandboxShowsServerError(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"POST /sandbox-uploads": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"internal_code":2007,"error_string":"TOKEN_TYPE_NOT_ALLOWED","status_code":403,"message":"You can only generate a sandbox upload url from a sandbox."}}`))
		},
	}

	stdout, stderr, err := runContractCommand(t, contractServer(t, routes), []string{"sandbox", "upload-url"}, nil, true, map[string]string{"path": "a.txt"})
	if err == nil {
		t.Fatalf("expected error, stdout=%q", stdout)
	}
	if !strings.Contains(err.Error(), "only generate a sandbox upload url from a sandbox") {
		t.Fatalf("err = %v", err)
	}
	assertNoSuccessJSON(t, stdout)
	assertNoTokenLeak(t, stdout, stderr, err)
}
