package utils

import (
	"encoding/json"
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

// WriteJSON marshals v to JSON and writes it to the command output.
func WriteJSON(cmd *cobra.Command, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}

// Hint writes a hint message to the command output with a subtle style indicator.
func Hint(cmd *cobra.Command, message string) {
	cmd.Println("→", message)
}
