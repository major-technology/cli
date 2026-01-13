package app

import (
	"os"
	"os/exec"
	"path/filepath"

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

	// Run start in current directory
	return RunStartInDir(cobraCmd, "")
}

// RunStartInDir changes to the specified directory and runs pnpm install and pnpm dev.
// If dir is empty, it uses the current directory.
func RunStartInDir(cmd *cobra.Command, dir string) error {
	// Change to the target directory if specified
	if dir != "" {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return errors.WrapError("failed to get directory path", err)
		}

		if err := os.Chdir(absDir); err != nil {
			return errors.WrapError("failed to change directory", err)
		}
		cmd.Printf("Changed to directory: %s\n", absDir)
	}

	// Run pnpm install
	cmd.Println("Running pnpm install...")
	installCmd := exec.Command("pnpm", "install")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	installCmd.Stdin = os.Stdin

	if err := installCmd.Run(); err != nil {
		return errors.WrapError("failed to run pnpm install", err)
	}

	cmd.Println("✓ Dependencies installed")

	// Run pnpm dev
	cmd.Println("\nStarting development server...")
	devCmd := exec.Command("pnpm", "dev")
	devCmd.Stdout = os.Stdout
	devCmd.Stderr = os.Stderr
	devCmd.Stdin = os.Stdin

	if err := devCmd.Run(); err != nil {
		return errors.WrapError("failed to run pnpm dev", err)
	}

	return nil
}
