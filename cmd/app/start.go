package app

import (
	"os"
	"os/exec"

	"github.com/major-technology/cli/errors"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the application locally",
	Long:  `Runs pnpm install and pnpm dev to set up dependencies and start the development server.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runStart(cobraCmd)
	},
}

func runStart(cobraCmd *cobra.Command) error {
	// Generate .env file
	_, _, err := generateEnvFile("")
	if err != nil {
		return errors.WrapError("failed to generate .env file", err)
	}

	// Run pnpm install
	cobraCmd.Println("Running pnpm install...")
	installCmd := exec.Command("pnpm", "install")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	installCmd.Stdin = os.Stdin

	if err := installCmd.Run(); err != nil {
		return errors.WrapError("failed to run pnpm install: %w", err)
	}

	cobraCmd.Println("✓ Dependencies installed")

	// Run pnpm dev
	cobraCmd.Println("\nStarting development server...")
	devCmd := exec.Command("pnpm", "dev")
	devCmd.Stdout = os.Stdout
	devCmd.Stderr = os.Stderr
	devCmd.Stdin = os.Stdin

	if err := devCmd.Run(); err != nil {
		return errors.WrapError("failed to run pnpm dev", err)
	}

	return nil
}
