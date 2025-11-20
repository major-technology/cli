package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the major CLI to the latest version",
	Long:  `Automatically detects your installation method (brew or direct install) and updates to the latest version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(cmd)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command) error {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87D7FF"))

	stepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87D7FF"))

	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00"))

	cmd.Println(titleStyle.Render("🔄 Updating Major CLI..."))
	cmd.Println()

	// Detect installation method
	installMethod := detectInstallMethod()

	cmd.Println(stepStyle.Render(fmt.Sprintf("▸ Detected installation method: %s", installMethod)))

	switch installMethod {
	case "brew":
		return updateViaBrew(cmd, stepStyle, successStyle)
	case "direct":
		return updateViaDirect(cmd, stepStyle, successStyle)
	default:
		return fmt.Errorf("could not detect installation method")
	}
}

func detectInstallMethod() string {
	// Check if installed via brew
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		// Check if brew is available
		if _, err := exec.LookPath("brew"); err == nil {
			// Check if major is installed via brew
			brewListCmd := exec.Command("brew", "list", "major")
			if err := brewListCmd.Run(); err == nil {
				return "brew"
			}
		}
	}

	// Otherwise assume direct install
	return "direct"
}

func updateViaBrew(cmd *cobra.Command, stepStyle, successStyle lipgloss.Style) error {
	cmd.Println(stepStyle.Render("▸ Updating via Homebrew..."))

	// Update brew first
	updateCmd := exec.Command("brew", "update")
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("failed to update Homebrew: %w", err)
	}

	// Upgrade major
	upgradeCmd := exec.Command("brew", "upgrade", "major")
	upgradeCmd.Stdout = os.Stdout
	upgradeCmd.Stderr = os.Stderr

	if err := upgradeCmd.Run(); err != nil {
		// Check if it's already up to date
		if strings.Contains(err.Error(), "already installed") {
			cmd.Println()
			cmd.Println(successStyle.Render("✓ Major CLI is already up to date!"))
			return nil
		}
		return fmt.Errorf("failed to upgrade major: %w", err)
	}

	cmd.Println()
	cmd.Println(successStyle.Render("✓ Successfully updated Major CLI!"))
	return nil
}

func updateViaDirect(cmd *cobra.Command, stepStyle, successStyle lipgloss.Style) error {
	cmd.Println(stepStyle.Render("▸ Downloading latest version..."))

	// Use the install script
	installScriptURL := "https://raw.githubusercontent.com/major-technology/cli/main/install.sh"

	// Download and execute the install script
	curlCmd := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", installScriptURL))
	curlCmd.Stdout = os.Stdout
	curlCmd.Stderr = os.Stderr
	curlCmd.Stdin = os.Stdin // Allow password prompt for sudo

	if err := curlCmd.Run(); err != nil {
		return fmt.Errorf("failed to download and install update: %w", err)
	}

	cmd.Println()
	cmd.Println(successStyle.Render("✓ Successfully updated Major CLI!"))
	return nil
}
