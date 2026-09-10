package git

import (
	"os/exec"
	"strings"
	"testing"
	"unicode"
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
	assertSingleBatchModeYes(t, sshCmd)
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
	if !strings.Contains(sshCmd, "/tmp/id_test") {
		t.Fatalf("GIT_SSH_COMMAND dropped existing identity: %q", sshCmd)
	}
	if got := sshOptionValues(sshCmd, "IdentitiesOnly"); len(got) != 1 || got[0] != "yes" {
		t.Fatalf("IdentitiesOnly = %q, want [yes] in %q", got, sshCmd)
	}
	assertSingleBatchModeYes(t, sshCmd)
	if envValue(cmd.Env, "GIT_TERMINAL_PROMPT") != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q, want 0", envValue(cmd.Env, "GIT_TERMINAL_PROMPT"))
	}
}

func TestConfigureRemoteCommandAddsBatchModeWhenSubstringLooksLikeOption(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "ssh -i /tmp/BatchMode=yes-key")
	SetNonInteractive(true)
	t.Cleanup(func() { SetNonInteractive(false) })

	cmd := exec.Command("git", "ls-remote")
	ConfigureRemoteCommand(cmd)

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	if !strings.Contains(sshCmd, "/tmp/BatchMode=yes-key") {
		t.Fatalf("GIT_SSH_COMMAND dropped identity path: %q", sshCmd)
	}
	assertSingleBatchModeYes(t, sshCmd)
}

func TestConfigureRemoteCommandOverridesConflictingBatchMode(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "ssh -o BatchMode=no")
	SetNonInteractive(true)
	t.Cleanup(func() { SetNonInteractive(false) })

	cmd := exec.Command("git", "ls-remote")
	ConfigureRemoteCommand(cmd)

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	assertSingleBatchModeYes(t, sshCmd)
}

func assertSingleBatchModeYes(t *testing.T, sshCmd string) {
	t.Helper()
	got := sshOptionValues(sshCmd, "BatchMode")
	if len(got) != 1 || got[0] != "yes" {
		t.Fatalf("BatchMode SSH options = %q, want [yes] in %q", got, sshCmd)
	}
}

func sshOptionValues(sshCmd, key string) []string {
	tokens := splitSSHArgs(sshCmd)
	var values []string
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "-o" {
			if i+1 >= len(tokens) {
				break
			}
			i++
			opt := tokens[i]
			k, v, ok := strings.Cut(opt, "=")
			if ok && strings.EqualFold(k, key) {
				values = append(values, v)
				continue
			}
			if strings.EqualFold(opt, key) && i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
				i++
				values = append(values, tokens[i])
			}
			continue
		}
		if len(tok) > 2 && strings.HasPrefix(tok, "-o") && !strings.HasPrefix(tok, "-o-") {
			rest := tok[2:]
			k, v, ok := strings.Cut(rest, "=")
			if ok && strings.EqualFold(k, key) {
				values = append(values, v)
			}
		}
	}
	return values
}

func splitSSHArgs(s string) []string {
	var tokens []string
	var b strings.Builder
	quote := rune(0)
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			if b.Len() > 0 {
				tokens = append(tokens, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens
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
