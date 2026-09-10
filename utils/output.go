package utils

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// WriteJSON encodes value as one JSON document on cmd stdout.
func WriteJSON(cmd *cobra.Command, value any) error {
	return json.NewEncoder(cmd.OutOrStdout()).Encode(value)
}

// Hint writes a next-command or diagnostic hint to cmd stderr.
func Hint(cmd *cobra.Command, text string) {
	fmt.Fprintln(cmd.ErrOrStderr(), text)
}
