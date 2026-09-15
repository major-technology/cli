package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestWriteJSONEncodesOneValueToStdout(t *testing.T) {
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := WriteJSON(cmd, map[string]any{"applicationId": "11111111-1111-4111-8111-111111111111"}); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var result map[string]any
	if err := decoder.Decode(&result); err != nil {
		t.Fatal(err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("unexpected trailing stdout: %v", err)
	}
	if result["applicationId"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("result = %#v", result)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestHintWritesToStderrNotStdout(t *testing.T) {
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	Hint(cmd, "major app logs --next-token cursor")

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if got := stderr.String(); got != "major app logs --next-token cursor\n" {
		t.Fatalf("stderr = %q", got)
	}
}
