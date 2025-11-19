package app

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// checkPnpmInstalled checks if pnpm is installed and provides installation instructions if not
func checkPnpmInstalled(cmd *cobra.Command) error {
	_, err := exec.LookPath("pnpm")
	if err != nil {
		errorStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5555"))

		commandStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#87D7FF"))

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5555")).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

		message := fmt.Sprintf("%s\n\n%s\n  %s\n\n%s\n  %s",
			errorStyle.Render("❌ pnpm is required for app commands"),
			"Install pnpm using one of these methods:",
			commandStyle.Render("brew install pnpm"),
			"Or if you have Node.js installed:",
			commandStyle.Render("corepack enable"))

		cmd.Println(boxStyle.Render(message))
		return fmt.Errorf("pnpm not found in PATH")
	}
	return nil
}

// Cmd represents the app command
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application management commands",
	Long:  `Commands for creating and managing applications.`,
	Args:  utils.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Check if pnpm is installed before running any app command
		if err := checkPnpmInstalled(cmd); err != nil {
			cobra.CheckErr(err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add app subcommands
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(infoCmd)
	Cmd.AddCommand(generateEnvCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(deployCmd)
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(cloneCmd)
}
