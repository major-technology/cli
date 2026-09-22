package vars

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/major-technology/cli/errors"
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
