package middleware

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/ui"
	"github.com/spf13/cobra"
)

// MiddlewareError is a custom error type that includes a title and suggestion
type MiddlewareError struct {
	Title      string
	Suggestion string
	Err        error
}

func (e *MiddlewareError) Error() string {
	return e.Title
}

func (e *MiddlewareError) Unwrap() error {
	return e.Err
}

// CommandCheck is a function that performs a check before a command is run
type CommandCheck func(cmd *cobra.Command, args []string) error

// Compose combines multiple checks into a single function compatible with Cobra's PreRunE
func Compose(checks ...CommandCheck) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, check := range checks {
			if err := check(cmd, args); err != nil {
				// If it's our custom MiddlewareError, print nicely
				if mwErr, ok := err.(*MiddlewareError); ok {
					ui.PrintError(cmd, mwErr.Title, mwErr.Suggestion)
					cmd.SilenceErrors = true
					cmd.SilenceUsage = true
					return err
				}

				// Fallback: print using ui.PrintError for consistency or rely on CheckErr if relevant
				// For checks that didn't return MiddlewareError but are simple errors:
				ui.PrintError(cmd, err.Error(), "")
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
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
	if err != nil {
		// Try to determine the specific error
		title := "Authentication failed"
		suggestion := "Run major user login to authenticate"

		if api.IsTokenExpired(err) || strings.Contains(err.Error(), "expired") {
			title = "Your session has expired!"
			suggestion = "Run major user login to login again."
		} else if api.IsNoToken(err) || strings.Contains(err.Error(), "not logged in") || strings.Contains(err.Error(), "invalid") {
			title = "Not logged in!"
			suggestion = "Run major user login to get started."
		}

		return &MiddlewareError{
			Title:      title,
			Suggestion: suggestion,
			Err:        err,
		}
	}

	return nil
}

// CheckPnpmInstalled checks if pnpm is installed in the system path
func CheckPnpmInstalled(cmd *cobra.Command, args []string) error {
	_, err := exec.LookPath("pnpm")
	if err != nil {
		return &MiddlewareError{
			Title:      "pnpm not found",
			Suggestion: "pnpm is required. Please install it: npm install -g pnpm",
			Err:        err,
		}
	}
	return nil
}

// CheckNodeVersion checks if node is installed and meets the minimum version requirement
func CheckNodeVersion(minVersion string) CommandCheck {
	return func(cmd *cobra.Command, args []string) error {
		path, err := exec.LookPath("node")
		if err != nil {
			return &MiddlewareError{
				Title:      "Node.js not found",
				Suggestion: "Node.js is required. Please install it.",
				Err:        err,
			}
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
			return &MiddlewareError{
				Title:      fmt.Sprintf("Node.js version %s required", minVersion),
				Suggestion: fmt.Sprintf("You are running version %s. Please upgrade Node.js.", versionStr),
				Err:        fmt.Errorf("node version %s is required, but found %s", minVersion, versionStr),
			}
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
