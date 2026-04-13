package vars

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagPullEnv  string
	flagPullFile string
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download environment variables to a local .env file",
	Long: `Download all environment variables for the selected environment into a
local dotenv file.

Writes both user-defined variables and platform-managed MAJOR_* system
variables needed for local development. Overwrites the target file.

If the target file is inside a git repository and is not yet ignored,
appends it to the repo's .gitignore.

Example:
  major vars pull --env staging --file .env.staging`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPull(cmd)
	},
}

func init() {
	pullCmd.Flags().StringVar(&flagPullEnv, "env", "", "Target environment name (defaults to your current environment)")
	pullCmd.Flags().StringVar(&flagPullFile, "file", ".env", "Path to write the dotenv file")
}

func runPull(cmd *cobra.Command) error {
	info, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	apiClient := singletons.GetAPIClient()

	// Resolve target environment (flag takes precedence over stored choice).
	env, err := resolveEnvironment(info.ApplicationID, flagPullEnv)
	if err != nil {
		return err
	}

	// If the user passed --env and it differs from their stored choice, switch
	// their stored choice to the requested environment before fetching. The
	// POST /application/env endpoint always uses the user's stored choice.
	if flagPullEnv != "" {
		currentResp, err := apiClient.GetApplicationEnvironment(info.ApplicationID)
		if err != nil {
			return errors.WrapError("failed to get current environment", err)
		}
		if currentResp.EnvironmentID == nil || *currentResp.EnvironmentID != env.ID {
			cmd.Printf("Setting your active environment to %q...\n", env.Name)
			if _, err := apiClient.SetApplicationEnvironment(info.ApplicationID, env.ID); err != nil {
				return errors.WrapError("failed to switch environment", err)
			}
		}
	}

	envVars, err := apiClient.GetApplicationEnv(info.OrganizationID, info.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to fetch environment variables", err)
	}

	// Sort keys: user-defined first (alphabetical), then MAJOR_* (alphabetical).
	userKeys := make([]string, 0, len(envVars))
	majorKeys := make([]string, 0)
	for k := range envVars {
		if strings.HasPrefix(k, "MAJOR_") {
			majorKeys = append(majorKeys, k)
		} else {
			userKeys = append(userKeys, k)
		}
	}
	sort.Strings(userKeys)
	sort.Strings(majorKeys)

	var builder strings.Builder
	fmt.Fprintf(&builder, "# Pulled from Major %q environment at %s\n", env.Name, time.Now().UTC().Format(time.RFC3339))
	builder.WriteString("# Do not edit MAJOR_* variables - they are managed by the platform\n\n")
	for _, k := range userKeys {
		builder.WriteString(formatDotenvLine(k, envVars[k]))
	}
	if len(majorKeys) > 0 && len(userKeys) > 0 {
		builder.WriteString("\n")
	}
	for _, k := range majorKeys {
		builder.WriteString(formatDotenvLine(k, envVars[k]))
	}

	targetPath, err := filepath.Abs(flagPullFile)
	if err != nil {
		return errors.WrapError("failed to resolve target file path", err)
	}

	if err := os.WriteFile(targetPath, []byte(builder.String()), 0600); err != nil {
		return errors.WrapError("failed to write dotenv file", err)
	}

	if err := ensureGitignore(cmd, targetPath); err != nil {
		// Non-fatal - warn but do not fail the pull.
		cmd.Printf("Warning: failed to update .gitignore: %v\n", err)
	}

	cmd.Printf("Environment: %s\n", env.Name)
	cmd.Printf("Pulled %d variables to %s.\n", len(envVars), flagPullFile)
	return nil
}

// formatDotenvLine returns a single KEY=value line with appropriate quoting.
func formatDotenvLine(key, value string) string {
	return fmt.Sprintf("%s=%s\n", key, quoteDotenvValue(value))
}

// quoteDotenvValue returns a dotenv-safe representation of value.
// Values containing whitespace or shell/special characters are double-quoted
// with embedded newlines, quotes, backslashes, and dollar signs escaped.
func quoteDotenvValue(value string) string {
	if value == "" {
		return ""
	}
	needsQuoting := false
	for _, r := range value {
		switch r {
		case ' ', '\t', '\n', '\r', '"', '\'', '`', '$', '#', '\\':
			needsQuoting = true
		}
		if needsQuoting {
			break
		}
	}
	if !needsQuoting {
		return value
	}

	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '$':
			b.WriteString(`\$`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// ensureGitignore appends the target file to .gitignore in the repo root if
// the file is inside a git repository and is not already ignored.
func ensureGitignore(cmd *cobra.Command, targetPath string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		// Not in a git repo - nothing to do.
		return nil
	}

	rel, err := filepath.Rel(repoRoot, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		// Target file is outside the repo - nothing to do.
		return nil
	}

	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	base := filepath.Base(targetPath)
	if isIgnored(existing, rel, base) {
		return nil
	}

	entry := rel
	// Normalize to forward slashes for gitignore.
	entry = filepath.ToSlash(entry)

	var out strings.Builder
	out.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		out.WriteString("\n")
	}
	out.WriteString(entry)
	out.WriteString("\n")

	if err := os.WriteFile(gitignorePath, []byte(out.String()), 0644); err != nil {
		return err
	}

	cmd.Printf("Added %s to .gitignore\n", entry)
	return nil
}

// isIgnored returns true if the provided .gitignore content already contains
// an entry that would match the target file (exact match, base name match,
// or a simple glob prefix match such as ".env*" for ".env").
func isIgnored(gitignoreContent, relPath, baseName string) bool {
	scanner := bufio.NewScanner(strings.NewReader(gitignoreContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "/")
		lineSlashed := filepath.ToSlash(line)
		relSlashed := filepath.ToSlash(relPath)
		if lineSlashed == relSlashed || lineSlashed == baseName {
			return true
		}
		// Simple prefix-glob support: ".env*" matches ".env" and ".env.local".
		if strings.HasSuffix(line, "*") {
			prefix := strings.TrimSuffix(line, "*")
			if strings.HasPrefix(baseName, prefix) {
				return true
			}
		}
	}
	return false
}
