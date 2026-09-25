package appclient

import (
	"os"
	"os/exec"
	"strings"

	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the app-client command
var Cmd = &cobra.Command{
	Use:   "app-client",
	Short: "Call other apps from this app",
	Long:  `Generate fetch clients that let the current application call other applications in its organization.`,
	Args:  utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(removeCmd)
}

// runAppClientCLI runs `pnpm exec major-app-client <args> [--framework <fw>]`.
// A variable so tests can stub the child process.
var runAppClientCLI = func(cmd *cobra.Command, dir string, args ...string) error {
	if framework := utils.DetectFramework(dir); framework != "" {
		args = append(args, "--framework", framework)
	}

	pnpmCmd := exec.Command("pnpm", append([]string{"exec", "major-app-client"}, args...)...)
	pnpmCmd.Dir = dir
	pnpmCmd.Stdout = cmd.ErrOrStderr()
	pnpmCmd.Stderr = cmd.ErrOrStderr()

	if err := pnpmCmd.Run(); err != nil {
		return errors.WrapError("major-app-client "+args[0]+" failed", err)
	}

	return nil
}

// ensureAppClientPackage adds @major-tech/app-client when package.json lacks it.
func ensureAppClientPackage(cmd *cobra.Command, dir string) error {
	data, err := os.ReadFile(dir + "/package.json")
	if err != nil {
		return errors.WrapError("failed to read package.json", err)
	}

	if strings.Contains(string(data), `"@major-tech/app-client"`) {
		return nil
	}

	addCmd := exec.Command("pnpm", "add", "@major-tech/app-client")
	addCmd.Dir = dir
	addCmd.Stdout = cmd.ErrOrStderr()
	addCmd.Stderr = cmd.ErrOrStderr()

	if err := addCmd.Run(); err != nil {
		return errors.WrapError("failed to install @major-tech/app-client", err)
	}

	return nil
}
