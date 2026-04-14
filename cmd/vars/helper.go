package vars

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
)

// keyPattern matches a valid env var key: must start with a letter or underscore,
// followed by any number of letters, digits, or underscores.
var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateKey checks that a key is a syntactically valid env var name and
// is not prefixed with MAJOR_ (reserved for platform-managed vars).
func validateKey(key string) error {
	if key == "" {
		return &errors.CLIError{
			Title:      "Key is required",
			Suggestion: "Pass a key name, e.g. DATABASE_URL",
		}
	}
	if !keyPattern.MatchString(key) {
		return &errors.CLIError{
			Title:      fmt.Sprintf("Invalid key: %q", key),
			Suggestion: "Keys must start with a letter or underscore and contain only letters, digits, or underscores.",
		}
	}
	if strings.HasPrefix(key, "MAJOR_") {
		return &errors.CLIError{
			Title:      fmt.Sprintf("Reserved key prefix: %q", key),
			Suggestion: "Keys starting with MAJOR_ are managed by the platform and cannot be set or unset from the CLI.",
		}
	}
	return nil
}

// resolvedEnv holds the target environment for a command invocation.
type resolvedEnv struct {
	ID   string
	Name string
}

// resolveEnvironment resolves the target environment for a command.
// If envFlag is non-empty, it looks up the environment by name (case-insensitive).
// Otherwise, it falls back to the user's currently-selected environment for the app.
func resolveEnvironment(applicationID, envFlag string) (*resolvedEnv, error) {
	apiClient := singletons.GetAPIClient()

	if envFlag != "" {
		listResp, err := apiClient.ListApplicationEnvironments(applicationID)
		if err != nil {
			return nil, errors.WrapError("failed to list environments", err)
		}
		lower := strings.ToLower(envFlag)
		for _, env := range listResp.Environments {
			if strings.ToLower(env.Name) == lower {
				return &resolvedEnv{ID: env.ID, Name: env.Name}, nil
			}
		}
		return nil, &errors.CLIError{
			Title:      fmt.Sprintf("Environment %q not found", envFlag),
			Suggestion: "Run 'major resource env' to see the environments available for this application.",
		}
	}

	envResp, err := apiClient.GetApplicationEnvironment(applicationID)
	if err != nil {
		return nil, errors.WrapError("failed to get current environment", err)
	}
	if envResp.EnvironmentID == nil || envResp.EnvironmentName == nil {
		return nil, &errors.CLIError{
			Title:      "No environment selected",
			Suggestion: "Run 'major resource env' to select an environment, or pass --env <name>.",
		}
	}
	return &resolvedEnv{ID: *envResp.EnvironmentID, Name: *envResp.EnvironmentName}, nil
}

// getAppID resolves the application ID for the current working directory.
func getAppID() (string, error) {
	info, err := utils.GetApplicationInfo("")
	if err != nil {
		return "", errors.WrapError("failed to identify application", err)
	}
	return info.ApplicationID, nil
}

// maskValue returns a masked representation of a value suitable for display.
// Values of 4 chars or fewer are fully masked; longer values show the first
// 4 characters followed by 8 bullets.
func maskValue(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("•", len(value))
	}
	return value[:4] + strings.Repeat("•", 8)
}

// findValueForEnv returns the value for a specific environment from a row's values slice,
// and whether it was found.
func findValueForEnv(values []api.EnvVariableValue, environmentID string) (string, bool) {
	for _, v := range values {
		if v.EnvironmentID == environmentID {
			return v.Value, true
		}
	}
	return "", false
}
