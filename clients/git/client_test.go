package git

import (
	"os/exec"
	"strings"
	"testing"
)

func TestConfigureRemoteCommandAddsBatchModeWhenSSHCommandUnset(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "")
	SetNonInteractive(true)
	t.Cleanup(func() { SetNonInteractive(false) })

	cmd := exec.Command("git", "ls-remote")
	ConfigureRemoteCommand(cmd)

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	if sshCmd == "" {
		t.Fatal("GIT_SSH_COMMAND must be set in non-interactive mode")
	}
	if !strings.Contains(sshCmd, "BatchMode=yes") {
		t.Fatalf("GIT_SSH_COMMAND = %q, want BatchMode=yes", sshCmd)
	}
	if envValue(cmd.Env, "GIT_TERMINAL_PROMPT") != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q, want 0", envValue(cmd.Env, "GIT_TERMINAL_PROMPT"))
	}
}

func TestConfigureRemoteCommandAddsBatchModeWhenSSHCommandAlreadySet(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "ssh -i /tmp/id_test -o IdentitiesOnly=yes")
	SetNonInteractive(true)
	t.Cleanup(func() { SetNonInteractive(false) })

	cmd := exec.Command("git", "ls-remote")
	ConfigureRemoteCommand(cmd)

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	if !strings.Contains(sshCmd, "ssh -i /tmp/id_test -o IdentitiesOnly=yes") {
		t.Fatalf("GIT_SSH_COMMAND dropped existing command: %q", sshCmd)
	}
	if !strings.Contains(sshCmd, "BatchMode=yes") {
		t.Fatalf("existing GIT_SSH_COMMAND must still include BatchMode=yes, got %q", sshCmd)
	}
	if envValue(cmd.Env, "GIT_TERMINAL_PROMPT") != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q, want 0", envValue(cmd.Env, "GIT_TERMINAL_PROMPT"))
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	found := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			found = strings.TrimPrefix(entry, prefix)
		}
	}
	return found
}
