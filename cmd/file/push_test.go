package file

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestKindForPath(t *testing.T) {
	cases := map[string]string{
		"notes.md":   "markdown",
		"Report.MD":  "markdown",
		"index.html": "html",
		"page.HTML":  "html",
	}
	for path, want := range cases {
		got, err := kindForPath(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if got != want {
			t.Fatalf("%s: kind = %q, want %q", path, got, want)
		}
	}

	if _, err := kindForPath("archive.zip"); err == nil {
		t.Fatal("expected error for .zip")
	}
}

func TestPrintPushResult(t *testing.T) {
	// Test with JSON=false: stdout should have link, stderr should have hint
	cmd := &cobra.Command{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	flagPushJSON = false
	err := printPushResult(cmd, "http://x/files/f1", "f1", 2)
	if err != nil {
		t.Fatalf("printPushResult failed: %v", err)
	}

	stdoutStr := stdout.String()
	if stdoutStr != "http://x/files/f1\n" {
		t.Fatalf("stdout = %q, want %q", stdoutStr, "http://x/files/f1\n")
	}

	stderrStr := stderr.String()
	if !bytes.Contains(stderr.Bytes(), []byte("f1")) {
		t.Fatalf("stderr should contain file id, got: %q", stderrStr)
	}

	// Test with JSON=true: stdout should be valid JSON, stderr should be empty
	stdout.Reset()
	stderr.Reset()

	flagPushJSON = true
	err = printPushResult(cmd, "http://x/files/f2", "f2", 1)
	if err != nil {
		t.Fatalf("printPushResult failed: %v", err)
	}

	stdoutStr = stdout.String()
	var result map[string]any
	if err := json.Unmarshal([]byte(stdoutStr), &result); err != nil {
		t.Fatalf("stdout is not valid JSON: %q, error: %v", stdoutStr, err)
	}

	if result["link"] != "http://x/files/f2" {
		t.Fatalf("link mismatch: %v", result["link"])
	}
	if result["fileId"] != "f2" {
		t.Fatalf("fileId mismatch: %v", result["fileId"])
	}
	if result["version"] != float64(1) {
		t.Fatalf("version mismatch: %v", result["version"])
	}

	stderrStr = stderr.String()
	if stderrStr != "" {
		t.Fatalf("stderr should be empty with JSON output, got: %q", stderrStr)
	}

	// Reset flagPushJSON
	flagPushJSON = false
}
