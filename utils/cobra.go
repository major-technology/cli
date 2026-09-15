package utils

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NoArgs returns an error with suggestions if any args are included.
// This is like cobra.NoArgs but includes command suggestions for typos.
func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		suggestions := cmd.SuggestionsFor(args[0])
		if len(suggestions) > 0 {
			return fmt.Errorf("unknown command %q for %q\n\nDid you mean this?\n\t%v", args[0], cmd.CommandPath(), suggestions[0])
		}
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return nil
}
