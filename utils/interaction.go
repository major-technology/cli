package utils

import (
	"fmt"
	"io"
	"os"

	xt "github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

// IsNonInteractive reports whether cmd must not prompt, open a browser, or
// wait on a TTY. True when --non-interactive is set or Cobra's input stream
// is not a terminal. --non-interactive never implies --yes.
func IsNonInteractive(cmd *cobra.Command) bool {
	if nonInteractiveFlag(cmd) {
		return true
	}
	return !inputIsTTY(cmd)
}

// RequireInteractive returns a concise error naming remedy when the command
// is running non-interactively. It is nil when prompts are allowed.
func RequireInteractive(cmd *cobra.Command, remedy string) error {
	if !IsNonInteractive(cmd) {
		return nil
	}
	if remedy == "" {
		return fmt.Errorf("this command requires an interactive terminal")
	}
	return fmt.Errorf("non-interactive: %s", remedy)
}

func nonInteractiveFlag(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup("non-interactive"); f != nil && f.Value.String() == "true" {
			return true
		}
		if f := c.PersistentFlags().Lookup("non-interactive"); f != nil && f.Value.String() == "true" {
			return true
		}
	}
	return false
}

func inputIsTTY(cmd *cobra.Command) bool {
	var in io.Reader = os.Stdin
	if cmd != nil {
		in = cmd.InOrStdin()
	}
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	return xt.IsTerminal(f.Fd())
}
