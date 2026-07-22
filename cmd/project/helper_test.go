package project

import (
	"errors"
	"testing"

	clierrors "github.com/major-technology/cli/errors"
)

// isProjectNotFoundError is a pure predicate over constructed CLIErrors, so
// it is tested directly here rather than through getProjectAndOrgID: that
// function's happy/error paths depend on git.GetRemoteURLFromDir shelling
// out to the real `git` binary in the current directory, and on the API
// client's keyring-backed auth (clients/api's testTokenOverride seam from T6
// is unexported and scoped to that package's own tests), so it cannot be
// driven end-to-end from an httptest server without real git/keyring state.
func TestIsProjectNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unmapped 404 CLIError",
			err:  &clierrors.CLIError{Title: "API Error (Code: 1002)", StatusCode: 404},
			want: true,
		},
		{
			name: "unmapped 401 CLIError",
			err:  &clierrors.CLIError{Title: "API Error (Code: 2000)", StatusCode: 401},
			want: false,
		},
		{
			name: "wrapped non-CLIError",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isProjectNotFoundError(tt.err); got != tt.want {
				t.Fatalf("isProjectNotFoundError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
