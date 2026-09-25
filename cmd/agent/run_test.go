package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/workspace"
	"github.com/spf13/cobra"
)

const testAgentID = "11111111-1111-4111-8111-111111111111"

func TestResolveAgentIDPrefersTheFlag(t *testing.T) {
	t.Chdir(t.TempDir())
	id, err := resolveAgentID("given")
	if err != nil || id != "given" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResolveAgentIDReadsTheAgentFolder(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "agent", AgentID: testAgentID}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	id, err := resolveAgentID("")
	if err != nil || id != testAgentID {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResolveAgentIDRejectsANonAgentFolder(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "app", ApplicationID: testAgentID}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	assertNeedsAgentID(t)
}

func TestResolveAgentIDRejectsAFolderWithNoConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	assertNeedsAgentID(t)
}

func assertNeedsAgentID(t *testing.T) {
	t.Helper()
	_, err := resolveAgentID("")
	if err == nil || !strings.Contains(err.Error(), "pass --id") || !strings.Contains(err.Error(), "agent folder") {
		t.Fatalf("err=%v", err)
	}
}

// executeNonInteractive runs cmd with empty stdin, so it must not prompt.
func executeNonInteractive(cmd *cobra.Command, args ...string) error {
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestSlackDeleteNeedsYesWhenNonInteractive(t *testing.T) {
	err := executeNonInteractive(newSlackCmd(), "delete", "--id", testAgentID)
	if err == nil || !strings.Contains(err.Error(), "requires --yes in non-interactive mode") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunNeedsAPrompt(t *testing.T) {
	err := executeNonInteractive(newRunCmd(), "--id", testAgentID)
	if err == nil || !strings.Contains(err.Error(), "--prompt") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunListRejectsAnUnknownSource(t *testing.T) {
	err := executeNonInteractive(newRunCmd(), "list", "--id", testAgentID, "--source", "bogus")
	if err == nil || !strings.Contains(err.Error(), "workflow") || !strings.Contains(err.Error(), "external_channel") {
		t.Fatalf("err=%v", err)
	}
}
