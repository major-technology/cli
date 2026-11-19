package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func writePackageJSON(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestClientProjectFindsViteHonoBackend(t *testing.T) {
	root := t.TempDir()
	writePackageJSON(t, root, `{"devDependencies":{"concurrently":"^10.0.0"}}`)
	writePackageJSON(t, filepath.Join(root, "backend"), `{"dependencies":{"hono":"^4.0.0"}}`)
	writePackageJSON(t, filepath.Join(root, "frontend"), `{"devDependencies":{"vite":"^8.0.0"}}`)

	clientDir := ClientProjectDir(root)
	if want := filepath.Join(root, "backend"); clientDir != want {
		t.Fatalf("ClientProjectDir = %q, want %q", clientDir, want)
	}
	if got := DetectFramework(clientDir); got != "hono" {
		t.Fatalf("DetectFramework = %q, want hono", got)
	}
}

func TestClientProjectKeepsNextAppAtRoot(t *testing.T) {
	root := t.TempDir()
	writePackageJSON(t, root, `{"dependencies":{"next":"16.0.0"}}`)

	clientDir := ClientProjectDir(root)
	if clientDir != root {
		t.Fatalf("ClientProjectDir = %q, want %q", clientDir, root)
	}
	if got := DetectFramework(clientDir); got != "nextjs" {
		t.Fatalf("DetectFramework = %q, want nextjs", got)
	}
}
