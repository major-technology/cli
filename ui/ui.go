package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// PrintError prints a styled error message to the command output
func PrintError(cmd *cobra.Command, title string, suggestion string) {
	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF5F87")).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5F87"))

	commandStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87D7FF"))

	var message string
	if suggestion != "" {
		message = fmt.Sprintf("%s\n\n%s", title, commandStyle.Render(suggestion))
	} else {
		message = title
	}

	cmd.Println(errorStyle.Render(message))
}
