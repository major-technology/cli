package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/errors"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:    "install",
	Short:  "Install the major CLI and setup shell integration",
	Hidden: true, // Internal command used by the installer script
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstall(cmd)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command) error {
	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00"))

	stepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87D7FF"))

	// Get the path to the current executable
	exe, err := os.Executable()
	if err != nil {
		return errors.WrapError("failed to get executable path", err)
	}

	// Resolve symlinks just in case
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return errors.WrapError("failed to resolve executable path", err)
	}

	binDir := filepath.Dir(exe)

	// Check for existing install in PATH that is different from current executable
	if pathMajor, err := exec.LookPath("major"); err == nil {
		// Resolve symlinks to compare real paths
		realPathMajor, _ := filepath.EvalSymlinks(pathMajor)
		if realPathMajor != "" && realPathMajor != exe {
			warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
			cmd.Println(warningStyle.Render(fmt.Sprintf("Warning: 'major' is already installed at %s", pathMajor)))
			cmd.Println("You may need to remove it or adjust your PATH to prioritize the new installation.")
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return errors.WrapError("failed to get user home dir", err)
	}

	shell := os.Getenv("SHELL")
	var configFile string
	var shellType string

	switch {
	case strings.Contains(shell, "zsh"):
		configFile = filepath.Join(home, ".zshrc")
		shellType = "zsh"
	case strings.Contains(shell, "bash"):
		configFile = filepath.Join(home, ".bashrc")
		// Check for .bash_profile on macOS
		if runtime.GOOS == "darwin" {
			if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
				configFile = filepath.Join(home, ".bash_profile")
			}
		}
		shellType = "bash"
	default:
		// Fallback or skip
		cmd.Println("Could not detect compatible shell (zsh/bash). Please add the following to your path manually:")
		cmd.Printf("  export PATH=\"%s:$PATH\"\n", binDir)
		return nil
	}

	// Create completions directory
	completionsDir := filepath.Join(home, ".major", "completions")
	if err := os.MkdirAll(completionsDir, 0755); err != nil {
		return errors.WrapError("failed to create completions directory", err)
	}

	// Generate completion script
	cmd.Println(stepStyle.Render("▸ Generating shell completions..."))

	var completionEntry string
	switch shellType {
	case "zsh":
		// For Zsh, we generate _major file and add directory to fpath
		completionFile := filepath.Join(completionsDir, "_major")
		f, err := os.Create(completionFile)
		if err != nil {
			return errors.WrapError("failed to create zsh completion file", err)
		}
		defer f.Close()

		if err := cmd.Root().GenZshCompletion(f); err != nil {
			return errors.WrapError("failed to generate zsh completion", err)
		}

		// We need to add fpath before compinit
		// But often users already have compinit in their .zshrc
		// The safest robust way is to append to fpath and ensure compinit is called
		completionEntry = fmt.Sprintf(`
# Major CLI
export PATH="%s:$PATH"
export FPATH="%s:$FPATH"
# Ensure compinit is loaded (if not already)
autoload -U compinit && compinit
`, binDir, completionsDir)

	case "bash":
		completionFile := filepath.Join(completionsDir, "major.bash")
		f, err := os.Create(completionFile)
		if err != nil {
			return errors.WrapError("failed to create bash completion file", err)
		}
		defer f.Close()

		if err := cmd.Root().GenBashCompletion(f); err != nil {
			return errors.WrapError("failed to generate bash completion", err)
		}

		completionEntry = fmt.Sprintf(`
# Major CLI
export PATH="%s:$PATH"
source "%s"
`, binDir, completionFile)
	}

	// Check if already configured
	content, err := os.ReadFile(configFile)
	if err == nil {
		// If we already see our marker or the bin path, we might want to update it or skip
		// But the user might have moved the directory.
		// Let's check for our specific comment
		if strings.Contains(string(content), "# Major CLI") {
			cmd.Println(successStyle.Render("Major CLI is already configured in your shell!"))
			// We still re-generated the completion file above, which is good for updates.
			return nil
		}
	}

	// Append to config
	f, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.WrapError("failed to open shell config file", err)
	}
	defer f.Close()

	if _, err := f.WriteString(completionEntry); err != nil {
		return errors.WrapError("failed to write to shell config file", err)
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FF00")).
		Padding(1, 2).
		MarginTop(1)

	msg := fmt.Sprintf("%s\n\nPlease restart your shell or run:\n\n  %s\n\nto start using 'major'",
		successStyle.Render(fmt.Sprintf("Added Major CLI configuration to %s", configFile)),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#87D7FF")).Render("source "+configFile))

	cmd.Println(boxStyle.Render(msg))

	return nil
}
