package transfer

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPresignedTransferDoesNotForwardCredential(t *testing.T) {
	data := []byte("archive bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("credential forwarded to presigned URL")
		}
		if r.Method == "PUT" {
			if r.Header.Get("Content-Type") != "application/zip" {
				t.Errorf("content type: %q", r.Header.Get("Content-Type"))
			}
			body, _ := io.ReadAll(r.Body)
			if !bytes.Equal(body, data) {
				t.Errorf("upload bytes: %q", body)
			}
			return
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	if err := Upload(server.URL, "application/zip", data); err != nil {
		t.Fatal(err)
	}
	got, err := Download(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("download: %q", got)
	}
}
