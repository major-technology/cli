package app

import (
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// getApplicationID retrieves the application ID for the current git repository
func getApplicationID() (string, error) {
	appID, _, _, err := getApplicationAndOrgIDFromDir("")
	return appID, err
}

// getApplicationAndOrgID retrieves the application ID, organization ID, and URL slug for the current git repository
func getApplicationAndOrgID() (string, string, string, error) {
	return getApplicationAndOrgIDFromDir("")
}

// getApplicationIDFromDir retrieves the application ID for a git repository in the specified directory.
// If dir is empty, it uses the current directory.
func getApplicationIDFromDir(dir string) (string, error) {
	appID, _, _, err := getApplicationAndOrgIDFromDir(dir)
	return appID, err
}

// getApplicationAndOrgIDFromDir retrieves the application ID, organization ID, and URL slug for a git repository in the specified directory.
// If dir is empty, it uses the current directory.
func getApplicationAndOrgIDFromDir(dir string) (string, string, string, error) {
	// Get the git remote URL from the specified directory
	remoteURL, err := git.GetRemoteURLFromDir(dir)
	if err != nil {
		return "", "", "", err
	}

	if remoteURL == "" {
		return "", "", "", fmt.Errorf("no git remote found in directory")
	}

	// Parse the remote URL to extract owner and repo
	remoteInfo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return "", "", "", errors.WrapError("failed to parse git remote URL", err)
	}

	// Get API client
	apiClient := singletons.GetAPIClient()
	if apiClient == nil {
		return "", "", "", fmt.Errorf("API client not initialized")
	}

	// Get application by repository
	appResp, err := apiClient.GetApplicationByRepo(remoteInfo.Owner, remoteInfo.Repo)
	if err != nil {
		return "", "", "", errors.WrapError("failed to get application", err)
	}

	var urlSlug string
	if appResp.URLSlug != nil {
		urlSlug = *appResp.URLSlug
	}

	return appResp.ApplicationID, appResp.OrganizationID, urlSlug, nil
}

// getPreferredCloneURL returns the preferred clone URL based on SSH availability
func getPreferredCloneURL(sshURL, httpsURL string) (url string, method string, err error) {
	if utils.CanUseSSH() && sshURL != "" {
		return sshURL, "SSH", nil
	}
	if httpsURL != "" {
		return httpsURL, "HTTPS", nil
	}
	return "", "", fmt.Errorf("no valid clone method available")
}

// ensureGitRepository ensures a directory is a properly configured git repository.
// If the directory doesn't exist, it clones the repo.
// If it exists but isn't a git repo, it initializes git and sets origin.
// If it exists and is a git repo, it ensures origin is set correctly and pulls.
// Returns the working directory path and any error.
func ensureGitRepository(cmd *cobra.Command, targetDir, sshURL, httpsURL string) error {
	cloneURL, cloneMethod, err := getPreferredCloneURL(sshURL, httpsURL)
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// Directory doesn't exist - clone fresh
		cmd.Printf("Cloning repository to '%s' using %s...\n", targetDir, cloneMethod)
		return git.Clone(cloneURL, targetDir)
	}

	// Directory exists - check if it's a git repo
	if !git.IsGitRepositoryDir(targetDir) {
		// Not a git repo - initialize and set origin
		cmd.Printf("Directory '%s' exists but is not a git repository. Initializing...\n", targetDir)
		if err := git.InitRepository(targetDir); err != nil {
			return errors.WrapError("failed to initialize git repository", err)
		}
	}

	// Ensure origin is set correctly
	cmd.Printf("Ensuring git origin is configured correctly...\n")
	if err := git.SetRemoteURL(targetDir, "origin", cloneURL); err != nil {
		return errors.WrapError("failed to set git origin", err)
	}

	// Pull latest changes
	cmd.Printf("Pulling latest changes...\n")
	return git.Pull(targetDir)
}

// cloneRepository clones a repository using SSH or HTTPS based on availability
// Returns the clone method used ("SSH" or "HTTPS") and any error
func cloneRepository(sshURL, httpsURL, targetDir string) (string, error) {
	// Determine which clone URL to use
	useSSH := false
	if utils.CanUseSSH() && sshURL != "" {
		useSSH = true
	} else if httpsURL == "" {
		return "", fmt.Errorf("no valid clone method available")
	}

	cloneURL := httpsURL
	cloneMethod := "HTTPS"
	if useSSH {
		cloneURL = sshURL
		cloneMethod = "SSH"
	}

	// Clone the repository
	if err := git.Clone(cloneURL, targetDir); err != nil {
		return "", errors.WrapError("failed to clone repository using "+cloneMethod, err)
	}

	return cloneMethod, nil
}

// isGitAuthError checks if the error is related to git authentication/permission issues
// It checks all wrapped errors in the chain, not just the top-level error
func isGitAuthError(err error) bool {
	if err == nil {
		return false
	}

	// Common git authentication error patterns
	authErrorPatterns := []string{
		"repository not found", // Catches "ERROR: Repository not found."
		"could not read from remote repository",
		"authentication failed",
		"permission denied",
		"403",
		"401",
		"access denied",
		"fatal: unable to access",
	}

	// Check all errors in the chain
	for e := err; e != nil; e = stderrors.Unwrap(e) {
		errMsg := strings.ToLower(e.Error())
		for _, pattern := range authErrorPatterns {
			if strings.Contains(errMsg, pattern) {
				return true
			}
		}
	}

	return false
}

// ensureGitRepositoryWithRetries retries ensureGitRepository with exponential backoff
func ensureGitRepositoryWithRetries(cmd *cobra.Command, workingDir, sshURL, httpsURL string) error {
	maxRetries := 3
	baseDelay := 200 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1)) // Exponential backoff: 2s, 4s, 8s, 16s, 32s
			cmd.Printf("Waiting %v for GitHub permissions to propagate...\n", delay)
			time.Sleep(delay)
		}

		err := ensureGitRepository(cmd, workingDir, sshURL, httpsURL)
		if err == nil {
			return nil
		}

		// If it's still an auth error, continue retrying
		if !isGitAuthError(err) {
			// Different error type, return immediately
			return err
		}
	}

	return errors.ErrorGitRepositoryAccessFailed
}

// generateEnvFile generates a .env file for the application in the specified directory.
// If targetDir is empty, it uses the current git repository root.
// Returns the path to the generated file and the env vars map.
func generateEnvFile(targetDir string) (string, map[string]string, error) {
	applicationID, orgID, _, err := getApplicationAndOrgIDFromDir(targetDir)
	if err != nil {
		return "", nil, errors.WrapError("failed to get application ID", err)
	}

	apiClient := singletons.GetAPIClient()

	envVars, err := apiClient.GetApplicationEnv(orgID, applicationID)
	if err != nil {
		return "", nil, errors.WrapError("failed to get environment variables", err)
	}

	gitRoot := targetDir
	if gitRoot == "" {
		gitRoot, err = git.GetRepoRoot()
		if err != nil {
			return "", nil, errors.WrapError("failed to get git repository root", err)
		}
	}

	// Create .env file path
	envFilePath := filepath.Join(gitRoot, ".env")

	// Build the .env file content
	var envContent strings.Builder
	for key, value := range envVars {
		envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Write to .env file
	err = os.WriteFile(envFilePath, []byte(envContent.String()), 0644)
	if err != nil {
		return "", nil, errors.WrapError("failed to write .env file", err)
	}

	return envFilePath, envVars, nil
}

// generateThemeFiles generates theme files (theme.css, theme.ts, logo.tsx) for the application.
// If targetDir is empty, it uses the current git repository root.
func generateThemeFiles(targetDir string) error {
	applicationID, _, _, err := getApplicationAndOrgIDFromDir(targetDir)
	if err != nil {
		return errors.WrapError("failed to get application ID", err)
	}

	apiClient := singletons.GetAPIClient()

	resp, err := apiClient.GetThemeFiles(applicationID)
	if err != nil {
		return errors.WrapError("failed to get theme files", err)
	}

	// No theme configured — not an error
	if resp.Css == nil && resp.ThemeModule == nil {
		return nil
	}

	gitRoot := targetDir
	if gitRoot == "" {
		gitRoot, err = git.GetRepoRoot()
		if err != nil {
			return errors.WrapError("failed to get git repository root", err)
		}
	}

	// Write theme.css
	if resp.Css != nil {
		cssPath := filepath.Join(gitRoot, "app", "theme.css")

		if err := os.MkdirAll(filepath.Dir(cssPath), 0755); err != nil {
			return errors.WrapError("failed to create app directory", err)
		}

		if err := os.WriteFile(cssPath, []byte(*resp.Css), 0644); err != nil {
			return errors.WrapError("failed to write theme.css", err)
		}
	}

	// Write lib/theme.ts
	if resp.ThemeModule != nil {
		modulePath := filepath.Join(gitRoot, "lib", "theme.ts")
		if err := os.MkdirAll(filepath.Dir(modulePath), 0755); err != nil {
			return errors.WrapError("failed to create lib directory", err)
		}

		if err := os.WriteFile(modulePath, []byte(*resp.ThemeModule), 0644); err != nil {
			return errors.WrapError("failed to write theme.ts", err)
		}
	}

	// Write components/ui/logo.tsx
	if resp.LogoComponent != nil {
		logoPath := filepath.Join(gitRoot, "components", "ui", "logo.tsx")
		if err := os.MkdirAll(filepath.Dir(logoPath), 0755); err != nil {
			return errors.WrapError("failed to create components/ui directory", err)
		}

		if err := os.WriteFile(logoPath, []byte(*resp.LogoComponent), 0644); err != nil {
			return errors.WrapError("failed to write logo.tsx", err)
		}
	}

	// Write theme skill for Claude Code
	if resp.Skill != nil {
		skillPath := filepath.Join(gitRoot, ".claude", "skills", "theme", "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
			return errors.WrapError("failed to create .claude/skills/theme directory", err)
		}

		if err := os.WriteFile(skillPath, []byte(*resp.Skill), 0644); err != nil {
			return errors.WrapError("failed to write theme skill", err)
		}
	}

	return nil
}

// handleThemeSync writes theme files on first-time setup, and prompts for upgrade
// if the app's theme version is behind the latest theme version.
func handleThemeSync(cmd *cobra.Command) error {
	applicationID, _, _, err := getApplicationAndOrgIDFromDir("")
	if err != nil {
		return err
	}

	gitRoot, err := git.GetRepoRoot()
	if err != nil {
		return err
	}

	existingCssPath := filepath.Join(gitRoot, "app", "theme.css")

	// First-time setup — write without prompting
	if _, statErr := os.Stat(existingCssPath); os.IsNotExist(statErr) {
		if err := generateThemeFiles(""); err != nil {
			return err
		}

		cmd.Println("✓ Theme files generated")
		return nil
	}

	// Check if upgrade is available
	apiClient := singletons.GetAPIClient()

	versionResp, err := apiClient.GetThemeVersion(applicationID)
	if err != nil {
		return nil // Non-fatal — skip version check
	}

	if !versionResp.UpgradeAvailable || versionResp.AppThemeVersion == nil || versionResp.LatestThemeVersion == nil {
		return nil
	}

	// Prompt for upgrade
	var confirm bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Theme upgrade available (v%d → v%d). Apply?",
					*versionResp.AppThemeVersion, *versionResp.LatestThemeVersion)).
				Value(&confirm),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if confirm {
		// Bump the version in the database first
		if err := apiClient.UpgradeTheme(applicationID); err != nil {
			return errors.WrapError("failed to upgrade theme", err)
		}

		// Then write theme files at the new version
		if err := generateThemeFiles(""); err != nil {
			return err
		}

		cmd.Println("✓ Theme files upgraded")
	}

	return nil
}

// renderColorBlock renders a colored █ block from a hex string, or plain █ if nil.
func renderColorBlock(hex *string) string {
	if hex == nil || *hex == "" {
		return "█"
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color(*hex)).Render("█")
}

// dimText renders text in gray for secondary info.
func dimText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(s)
}

// buildThemeSelectField builds a huh.Select field for theme selection.
// Returns the field and a pointer to the selected value. Returns nil if no themes available.
func buildThemeSelectField(apiClient *api.Client, orgID string, selectedID *string) (huh.Field, error) {
	resp, err := apiClient.ListThemes(orgID)
	if err != nil {
		return nil, err
	}

	if len(resp.Themes) == 0 {
		return nil, nil
	}

	options := make([]huh.Option[string], 0, len(resp.Themes)+1)

	// "Use default" option first, pre-selected
	var defaultThemeName string

	for _, t := range resp.Themes {
		if t.IsDefault {
			defaultThemeName = t.Name
			*selectedID = t.ID
			break
		}
	}

	if defaultThemeName != "" {
		options = append(options, huh.NewOption("Use default ("+defaultThemeName+")", *selectedID))
	}

	for _, t := range resp.Themes {
		if t.IsDefault && defaultThemeName != "" {
			continue // Already added as "Use default" option
		}

		var baseSwatch, accentSwatch string

		if t.DisplayColors != nil {
			baseSwatch = renderColorBlock(t.DisplayColors.BaseColorHex)
			accentSwatch = renderColorBlock(t.DisplayColors.AccentColorHex)
		} else {
			baseSwatch = "█"
			accentSwatch = "█"
		}

		label := t.Name
		label += "  " + dimText("Base") + " " + baseSwatch
		label += "  " + dimText("Accent") + " " + accentSwatch
		label += "  " + dimText("Font:") + " " + t.Config.Font
		label += "  " + dimText("Radius:") + " " + t.Config.Radius

		if t.Config.Elevation != "" {
			label += "  " + dimText("Elevation:") + " " + t.Config.Elevation
		}

		options = append(options, huh.NewOption(label, t.ID))
	}

	field := huh.NewSelect[string]().
		Title("Theme").
		Description("Select a theme for your application").
		Options(options...).
		Height(5).
		Value(selectedID)

	return field, nil
}
