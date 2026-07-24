package project

import (
	"strings"
	"testing"
)

func TestCreateRequiresName(t *testing.T) {
	out, err := runCommand(t, newCreateCmd())
	if err == nil {
		t.Fatalf("expected arg error, got success: %s", out)
	}
	if !strings.Contains(err.Error(), "arg") {
		t.Fatalf("expected missing-arg error, got: %v", err)
	}
}
