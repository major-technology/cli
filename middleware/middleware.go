package middleware

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

// CommandCheck is a function that performs a check before a command is run
type CommandCheck func(cmd *cobra.Command, args []string) error

// Compose combines multiple checks into a single function compatible with Cobra's PreRunE/PersistentPreRunE
// All errors are returned without printing - error formatting is handled centrally in Execute()
func Compose(checks ...CommandCheck) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, check := range checks {
			if err := check(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

// ChainParent combines the parent command's PersistentPreRunE with additional checks
// This ensures child commands inherit parent middleware while adding their own
// All errors are returned without printing - error formatting is handled centrally in Execute()
func ChainParent(checks ...CommandCheck) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// First, call parent's PersistentPreRunE if it exists
		parent := cmd.Parent()
		if parent != nil && parent.PersistentPreRunE != nil {
			if err := parent.PersistentPreRunE(parent, args); err != nil {
				return err
			}
		}

		// Then run the additional checks
		for _, check := range checks {
			if err := check(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

// CheckLogin checks if the user is logged in and the session is valid
func CheckLogin(cmd *cobra.Command, args []string) error {
	client := singletons.GetAPIClient()

	// VerifyToken checks if the token exists and is valid by calling the API
	_, err := client.VerifyToken()
	return err
}

// CheckVersion checks if the CLI version is up to date and handles upgrade prompts
func CheckVersion(version string) CommandCheck {
	return func(cmd *cobra.Command, args []string) error {
		// Skip for dev version
		if version == "dev" {
			return nil
		}

		// Skip for update command itself
		if cmd.Name() == "update" {
			return nil
		}

		client := singletons.GetAPIClient()

		resp, err := client.CheckVersion(version)
		if err != nil {
			// Silently ignore version check errors to not disrupt user workflow
			return nil
		}

		// Check for force upgrade
		if resp.ForceUpgrade {
			return clierrors.ErrorForceUpgrade
		}

		// Check for optional upgrade
		if resp.CanUpgrade {
			warningStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFD700"))

			commandStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#87D7FF"))

			message := fmt.Sprintf("%s %s",
				warningStyle.Render("There's a new version of major available."),
				fmt.Sprintf("Run %s to get the newest version.",
					commandStyle.Render("major update")))

			cmd.Println(message)
			cmd.Println() // Add a blank line for spacing
		}

		return nil
	}
}

// CheckNodeInstalled checks if node is installed in the system path
func CheckNodeInstalled(cmd *cobra.Command, args []string) error {
	_, err := exec.LookPath("node")
	if err != nil {
		return clierrors.ErrorNodeNotFound
	}
	return nil
}

// CheckPnpmInstalled checks if pnpm is installed in the system path
func CheckPnpmInstalled(cmd *cobra.Command, args []string) error {
	_, err := exec.LookPath("pnpm")
	if err != nil {
		return clierrors.ErrorPnpmNotFound
	}
	return nil
}

// CheckNodeVersion checks if node is installed and meets the minimum version requirement
func CheckNodeVersion(minVersion string) CommandCheck {
	return func(cmd *cobra.Command, args []string) error {
		path, err := exec.LookPath("node")
		if err != nil {
			return clierrors.ErrorNodeNotFound
		}

		cmdOut := exec.Command(path, "--version")
		output, err := cmdOut.Output()
		if err != nil {
			return fmt.Errorf("failed to check node version: %w", err)
		}

		versionStr := strings.TrimSpace(string(output))
		// Remove 'v' prefix if present
		versionStr = strings.TrimPrefix(versionStr, "v")

		if !isVersionGTE(versionStr, minVersion) {
			return clierrors.ErrorNodeVersionTooOld(minVersion, versionStr)
		}

		return nil
	}
}

// isVersionGTE returns true if v1 >= v2
// This is a simple implementation assuming semver format x.y.z
func isVersionGTE(v1, v2 string) bool {
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	for i := 0; i < 3; i++ {
		if parts1[i] > parts2[i] {
			return true
		}
		if parts1[i] < parts2[i] {
			return false
		}
	}
	return true
}

func parseVersion(v string) [3]int {
	var parts [3]int

	// Use regex to extract numbers
	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(v)

	if len(matches) == 4 {
		for i := 1; i <= 3; i++ {
			val, _ := strconv.Atoi(matches[i])
			parts[i-1] = val
		}
	} else {
		// Fallback for simpler versions like "18" or "18.1"
		split := strings.Split(v, ".")
		for i := 0; i < len(split) && i < 3; i++ {
			val, _ := strconv.Atoi(split[i])
			parts[i] = val
		}
	}

	return parts
}
