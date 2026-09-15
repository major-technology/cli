package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestCreateFileRequestShape(t *testing.T) {
	srv, c := newTestServer(t, http.MethodPost, "/files", http.StatusOK, HostedFileResponse{
		File: HostedFile{ID: "f1", Name: "report.md", Kind: "markdown", Version: 1, Link: "http://x/files/f1"},
	})
	defer srv.Close()

	resp, err := c.CreateFile("org-1", "report.md", "markdown", "# hi")
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if resp.File.Link != "http://x/files/f1" || resp.File.Version != 1 {
		t.Fatalf("unexpected response: %+v", resp.File)
	}
}

func TestCreateFileRequestBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var body createHostedFileRequest
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if body.OrganizationID != "org-1" || body.Name != "report.md" || body.Kind != "markdown" || body.Content != "# hi" {
			t.Errorf("body = %+v, want org-1/report.md/markdown/# hi", body)
		}

		assertJSONKeys(t, data, "organizationId", "name", "kind", "content")

		_ = json.NewEncoder(w).Encode(HostedFileResponse{File: HostedFile{ID: "f1"}})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	if _, err := client.CreateFile("org-1", "report.md", "markdown", "# hi"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
}

func TestPushFileVersionPath(t *testing.T) {
	srv, c := newTestServer(t, http.MethodPost, "/files/f1/versions", http.StatusOK, HostedFileResponse{
		File: HostedFile{ID: "f1", Version: 2},
	})
	defer srv.Close()

	resp, err := c.PushFileVersion("f1", "markdown", "# v2")
	if err != nil {
		t.Fatalf("PushFileVersion: %v", err)
	}
	if resp.File.Version != 2 {
		t.Fatalf("version = %d, want 2", resp.File.Version)
	}
}

func TestPushFileVersionRequestBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var body pushHostedFileVersionRequest
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if body.Kind != "markdown" || body.Content != "# v2" {
			t.Errorf("body = %+v, want markdown/# v2", body)
		}

		assertJSONKeys(t, data, "kind", "content")

		_ = json.NewEncoder(w).Encode(HostedFileResponse{File: HostedFile{ID: "f1", Version: 2}})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	if _, err := client.PushFileVersion("f1", "markdown", "# v2"); err != nil {
		t.Fatalf("PushFileVersion: %v", err)
	}
}

// assertJSONKeys decodes data as a JSON object and fails the test unless its
// key set is exactly wantKeys, so a request-tag typo (extra/missing/renamed
// field) is caught even when the values happen to round-trip.
func assertJSONKeys(t *testing.T, data []byte, wantKeys ...string) {
	t.Helper()

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("bad body: %v", err)
	}

	got := make([]string, 0, len(raw))
	for k := range raw {
		got = append(got, k)
	}
	sort.Strings(got)
	want := append([]string(nil), wantKeys...)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("keys = %v, want %v", got, want)
		}
	}
}

func TestGetFileContentURLPath(t *testing.T) {
	srv, c := newTestServer(t, http.MethodGet, "/files/f1/content-url", http.StatusOK, HostedFileContentURLResponse{
		Name: "report.md", Kind: "markdown", Version: 2, URL: "https://s3/x",
	})
	defer srv.Close()

	resp, err := c.GetFileContentURL("f1")
	if err != nil {
		t.Fatalf("GetFileContentURL: %v", err)
	}
	if resp.URL != "https://s3/x" {
		t.Fatalf("url = %q", resp.URL)
	}
}

func TestListFilesQuery(t *testing.T) {
	srv, c := newTestServer(t, http.MethodGet, "/files?organizationId=org-1", http.StatusOK, ListHostedFilesResponse{
		Files: []HostedFile{{ID: "f1"}},
	})
	defer srv.Close()

	resp, err := c.ListFiles("org-1")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(resp.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(resp.Files))
	}
}

func TestRenameDeleteSharePaths(t *testing.T) {
	srv, c := newTestServer(t, http.MethodPatch, "/files/f1", http.StatusOK, HostedFileResponse{File: HostedFile{Name: "new.md"}})
	if _, err := c.RenameFile("f1", "new.md"); err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	srv.Close()

	srv, c = newTestServer(t, http.MethodDelete, "/files/f1", http.StatusOK, map[string]bool{"success": true})
	if err := c.DeleteFile("f1"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	srv.Close()

	srv, c = newTestServer(t, http.MethodPost, "/files/f1/share-by-email", http.StatusOK, map[string]bool{"success": true})
	if err := c.ShareFileByEmail("f1", "a@b.com", "File:Viewer"); err != nil {
		t.Fatalf("ShareFileByEmail: %v", err)
	}
	srv.Close()
}

func TestFileEndpointsErrorMapping(t *testing.T) {
	srv, c := newTestServer(t, http.MethodPost, "/files/f1/versions", http.StatusForbidden, ErrorResponse{
		Error: &AppErrorDetail{InternalCode: 9999, ErrorString: "You do not have file:edit permission for this file", StatusCode: 403},
	})
	defer srv.Close()

	_, err := c.PushFileVersion("f1", "markdown", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnmappedErrorUsesServerErrorString(t *testing.T) {
	srv, c := newTestServer(t, http.MethodPost, "/files/f1/versions", http.StatusBadRequest, ErrorResponse{
		Error: &AppErrorDetail{InternalCode: 9999, ErrorString: "File is markdown; cannot push html content onto it", StatusCode: 400},
	})
	defer srv.Close()

	_, err := c.PushFileVersion("f1", "html", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "File is markdown; cannot push html content onto it" {
		t.Fatalf("Error() = %q, want server error string", err.Error())
	}
}
