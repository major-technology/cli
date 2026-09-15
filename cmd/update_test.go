package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func TestUpdateDirectNonInteractiveRefusesPrivilegeEscalation(t *testing.T) {
	started := 0
	var stdinAttached *os.File
	orig := runDirectInstall
	runDirectInstall = func(stdin *os.File) error {
		started++
		stdinAttached = stdin
		return nil
	}
	t.Cleanup(func() { runDirectInstall = orig })

	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	if err := root.PersistentFlags().Set("non-interactive", "true"); err != nil {
		t.Fatal(err)
	}
	child := &cobra.Command{Use: "update"}
	root.AddCommand(child)
	child.SetIn(strings.NewReader(""))
	out := &bytes.Buffer{}
	child.SetOut(out)
	child.SetErr(out)

	err := runWithDeadline(t, 8*time.Second, func() error {
		return updateViaDirect(child, lipgloss.NewStyle(), lipgloss.NewStyle())
	})
	if err == nil {
		t.Fatal("non-interactive update must refuse sudo/password prompts")
	}
	if !strings.Contains(err.Error(), "interactive") {
		t.Fatalf("error must name interactive remedy, got %v", err)
	}
	if started != 0 {
		t.Fatalf("direct install started = %d stdin=%v, want 0 (must not attach os.Stdin)", started, stdinAttached)
	}
}
