package utils

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireInteractiveRefusesNonTTYInput(t *testing.T) {
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)
	child.SetIn(strings.NewReader(""))
	if err := RequireInteractive(child, "Pass --yes to confirm deletion."); err == nil {
		t.Fatal("non-TTY input must refuse confirmation")
	}
}

func TestRequireInteractiveErrorNamesRemedy(t *testing.T) {
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)
	child.SetIn(strings.NewReader(""))
	err := RequireInteractive(child, "Pass --yes to confirm deletion.")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Pass --yes to confirm deletion.") {
		t.Fatalf("error = %v, want remedy named", err)
	}
}

func TestIsNonInteractiveHonorsFlag(t *testing.T) {
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)
	if err := root.PersistentFlags().Set("non-interactive", "true"); err != nil {
		t.Fatal(err)
	}
	if !IsNonInteractive(child) {
		t.Fatal("explicit --non-interactive must be non-interactive")
	}
}

func TestIsNonInteractiveTreatsEmptyReaderAsNonTTY(t *testing.T) {
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)
	child.SetIn(strings.NewReader(""))
	if !IsNonInteractive(child) {
		t.Fatal("Cobra input stream that is not a TTY must be non-interactive")
	}
}

func TestRequireInteractiveAllowsTTYWithoutFlag(t *testing.T) {
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)
	// A pipe is not a TTY; this test only applies when stdin is a real terminal.
	// When the command input is an *os.File terminal, RequireInteractive must allow prompts.
	if !IsNonInteractive(child) {
		if err := RequireInteractive(child, "Pass --yes to confirm deletion."); err != nil {
			t.Fatalf("interactive TTY must allow prompts: %v", err)
		}
	}
}
