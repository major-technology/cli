package api

import (
	"net/http"
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

func TestPushFileVersionPath(t *testing.T) {
	srv, c := newTestServer(t, http.MethodPost, "/files/f1/versions", http.StatusOK, HostedFileResponse{
		File: HostedFile{ID: "f1", Version: 2},
	})
	defer srv.Close()

	resp, err := c.PushFileVersion("f1", "# v2")
	if err != nil {
		t.Fatalf("PushFileVersion: %v", err)
	}
	if resp.File.Version != 2 {
		t.Fatalf("version = %d, want 2", resp.File.Version)
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

	_, err := c.PushFileVersion("f1", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}
