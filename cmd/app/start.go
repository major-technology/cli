package app

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the application development server",
	Long:  `Runs pnpm install and pnpm dev to set up dependencies and start the development server.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runStart(cobraCmd))
	},
}

func runStart(cobraCmd *cobra.Command) error {
	// Check if pnpm is installed
	if _, err := exec.LookPath("pnpm"); err != nil {
		return fmt.Errorf("pnpm is not installed. Please install pnpm first: https://pnpm.io/installation")
	}

	// Run pnpm install
	cobraCmd.Println("Running pnpm install...")
	installCmd := exec.Command("pnpm", "install")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	installCmd.Stdin = os.Stdin

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to run pnpm install: %w", err)
	}

	cobraCmd.Println("✓ Dependencies installed")

	// Run pnpm dev
	cobraCmd.Println("\nStarting development server...")
	devCmd := exec.Command("pnpm", "dev")
	devCmd.Stdout = os.Stdout
	devCmd.Stderr = os.Stderr
	devCmd.Stdin = os.Stdin

	if err := devCmd.Run(); err != nil {
		return fmt.Errorf("failed to run pnpm dev: %w", err)
	}

	return nil
}

