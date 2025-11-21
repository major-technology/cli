package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	// Get the path to the current executable
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks just in case
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
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

	// If we are in a temp directory or not in the expected location,
	// we might want to copy ourselves?
	// The script does the downloading, so we assume we are already in ~/.major/bin/major
	// We just need to ensure ~/.major/bin is in PATH

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}

	shell := os.Getenv("SHELL")
	var configFile string

	switch {
	case strings.Contains(shell, "zsh"):
		configFile = filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		configFile = filepath.Join(home, ".bashrc")
		// Check for .bash_profile on macOS
		if runtime.GOOS == "darwin" {
			if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
				configFile = filepath.Join(home, ".bash_profile")
			}
		}
	default:
		// Fallback or skip
		cmd.Println("Could not detect compatible shell (zsh/bash). Please add the following to your path manually:")
		cmd.Printf("  export PATH=\"%s:$PATH\"\n", binDir)
		return nil
	}

	// Check if already in config
	content, err := os.ReadFile(configFile)
	if err == nil {
		if strings.Contains(string(content), binDir) {
			cmd.Println(successStyle.Render("Major CLI is already in your PATH!"))
			return nil
		}
	}

	// Append to config
	f, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open shell config file: %w", err)
	}
	defer f.Close()

	pathEntry := fmt.Sprintf("\n# Major CLI\nexport PATH=\"%s:$PATH\"\n", binDir)
	if _, err := f.WriteString(pathEntry); err != nil {
		return fmt.Errorf("failed to write to shell config file: %w", err)
	}

	cmd.Println(successStyle.Render(fmt.Sprintf("Added Major CLI to %s", configFile)))
	cmd.Println("Please restart your shell or source your config file to start using 'major'")

	return nil
}
