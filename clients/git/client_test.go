package git

import (
	"os/exec"
	"strings"
	"testing"
)

func TestConfigureRemoteCommandAddsBatchModeWhenSSHCommandUnset(t *testing.T) {
	cmd, err := configureRemote(t, "")
	if err != nil {
		t.Fatal(err)
	}

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	if sshCmd == "" {
		t.Fatal("GIT_SSH_COMMAND must be set in non-interactive mode")
	}
	assertEffectiveBatchModeYes(t, sshCmd)
	if envValue(cmd.Env, "GIT_TERMINAL_PROMPT") != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q, want 0", envValue(cmd.Env, "GIT_TERMINAL_PROMPT"))
	}
}

func TestConfigureRemoteCommandAddsBatchModeWhenSSHCommandAlreadySet(t *testing.T) {
	cmd, err := configureRemote(t, "ssh -i /tmp/id_test -o IdentitiesOnly=yes")
	if err != nil {
		t.Fatal(err)
	}

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	if !strings.Contains(sshCmd, "/tmp/id_test") {
		t.Fatalf("GIT_SSH_COMMAND dropped existing identity: %q", sshCmd)
	}
	if got := sshOptionValuesFromArgv(shellArgv(t, sshCmd), "IdentitiesOnly"); len(got) != 1 || got[0] != "yes" {
		t.Fatalf("IdentitiesOnly = %q, want [yes] in %q", got, sshCmd)
	}
	assertEffectiveBatchModeYes(t, sshCmd)
	if envValue(cmd.Env, "GIT_TERMINAL_PROMPT") != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q, want 0", envValue(cmd.Env, "GIT_TERMINAL_PROMPT"))
	}
}

func TestConfigureRemoteCommandAddsBatchModeWhenSubstringLooksLikeOption(t *testing.T) {
	cmd, err := configureRemote(t, "ssh -i /tmp/BatchMode=yes-key")
	if err != nil {
		t.Fatal(err)
	}

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	if !strings.Contains(sshCmd, "/tmp/BatchMode=yes-key") {
		t.Fatalf("GIT_SSH_COMMAND dropped identity path: %q", sshCmd)
	}
	assertEffectiveBatchModeYes(t, sshCmd)
}

func TestConfigureRemoteCommandOverridesConflictingBatchMode(t *testing.T) {
	cmd, err := configureRemote(t, "ssh -o BatchMode=no")
	if err != nil {
		t.Fatal(err)
	}

	sshCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
	assertEffectiveBatchModeYes(t, sshCmd)
}

func TestConfigureRemoteCommandPreservesQuotedAndEscapedIdentityPaths(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantIdentity    string
		mustKeepLiteral string
	}{
		{
			name:            "single-quoted identity",
			input:           `ssh -i '/tmp/key file'`,
			wantIdentity:    "/tmp/key file",
			mustKeepLiteral: `'/tmp/key file'`,
		},
		{
			name:            "double-quoted identity",
			input:           `ssh -i "/tmp/key file"`,
			wantIdentity:    "/tmp/key file",
			mustKeepLiteral: `"/tmp/key file"`,
		},
		{
			name:            "backslash-escaped identity",
			input:           `ssh -i /tmp/key\ file`,
			wantIdentity:    "/tmp/key file",
			mustKeepLiteral: `/tmp/key\ file`,
		},
		{
			name:            "quoted identity with other options",
			input:           `ssh -i '/tmp/key file' -o IdentitiesOnly=yes`,
			wantIdentity:    "/tmp/key file",
			mustKeepLiteral: `'/tmp/key file'`,
		},
		{
			name:            "escaped double quote in identity",
			input:           `ssh -i "/tmp/key\"file"`,
			wantIdentity:    `/tmp/key"file`,
			mustKeepLiteral: `"/tmp/key\"file"`,
		},
		{
			name:            "single-quoted dollar and space",
			input:           `ssh -i '$HOME/key file'`,
			wantIdentity:    "$HOME/key file",
			mustKeepLiteral: `'$HOME/key file'`,
		},
		{
			name:            "double-quoted proxy command",
			input:           `ssh -i '/tmp/key file' -o ProxyCommand="nc %h %p"`,
			wantIdentity:    "/tmp/key file",
			mustKeepLiteral: `ProxyCommand="nc %h %p"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if identityFlagValue(shellArgv(t, tt.input), "-i") != tt.wantIdentity {
				t.Fatalf("fixture %q did not parse as identity %q", tt.input, tt.wantIdentity)
			}
			cmd, err := configureRemote(t, tt.input)
			if err != nil {
				t.Fatal(err)
			}

			gotCmd := envValue(cmd.Env, "GIT_SSH_COMMAND")
			if !strings.Contains(gotCmd, tt.mustKeepLiteral) {
				t.Fatalf("rewrote original quoting, lost %q in %q", tt.mustKeepLiteral, gotCmd)
			}
			got := shellArgv(t, gotCmd)
			if !containsArg(got, tt.wantIdentity) {
				t.Fatalf("shell argv lost identity %q: %#v (rewritten %q)", tt.wantIdentity, got, gotCmd)
			}
			assertEffectiveBatchModeYes(t, gotCmd)
			if identityFlagValue(got, "-i") != tt.wantIdentity {
				t.Fatalf("-i value = %q, want %q in %#v (rewritten %q)", identityFlagValue(got, "-i"), tt.wantIdentity, got, gotCmd)
			}
		})
	}
}

func TestConfigureRemoteCommandRejectsNonSSHFirstExecutable(t *testing.T) {
	cmd, err := configureRemote(t, "env GIT_SSH_VARIANT=1 ssh -i /tmp/id")
	if err == nil {
		t.Fatal("wrapper/env GIT_SSH_COMMAND must be rejected in non-interactive mode")
	}
	assertUnsupportedGITSSHCommand(t, err)
	if envValue(cmd.Env, "GIT_SSH_COMMAND") != "" {
		t.Fatalf("must not rewrite unsupported GIT_SSH_COMMAND, got %q", envValue(cmd.Env, "GIT_SSH_COMMAND"))
	}
}

func TestConfigureRemoteCommandRejectsCompoundCommand(t *testing.T) {
	cmd, err := configureRemote(t, "ssh -i /tmp/id; echo pwned")
	if err == nil {
		t.Fatal("compound GIT_SSH_COMMAND must be rejected in non-interactive mode")
	}
	assertUnsupportedGITSSHCommand(t, err)
	if envValue(cmd.Env, "GIT_SSH_COMMAND") != "" {
		t.Fatalf("must not rewrite unsupported GIT_SSH_COMMAND, got %q", envValue(cmd.Env, "GIT_SSH_COMMAND"))
	}
}

func configureRemote(t *testing.T, gitSSHCommand string) (*exec.Cmd, error) {
	t.Helper()
	t.Setenv("GIT_SSH_COMMAND", gitSSHCommand)
	SetNonInteractive(true)
	t.Cleanup(func() { SetNonInteractive(false) })
	cmd := exec.Command("git", "ls-remote")
	err := ConfigureRemoteCommand(cmd)
	return cmd, err
}

func assertUnsupportedGITSSHCommand(t *testing.T, err error) {
	t.Helper()
	msg := err.Error()
	if !strings.Contains(msg, "GIT_SSH_COMMAND") {
		t.Fatalf("error must name GIT_SSH_COMMAND, got %v", err)
	}
	if !strings.Contains(msg, "direct ssh") {
		t.Fatalf("error must name supported direct ssh syntax, got %v", err)
	}
}

func assertEffectiveBatchModeYes(t *testing.T, sshCmd string) {
	t.Helper()
	got := sshOptionValuesFromArgv(shellArgv(t, sshCmd), "BatchMode")
	if len(got) == 0 || got[0] != "yes" {
		t.Fatalf("effective BatchMode = %q, want first value yes in %q", got, sshCmd)
	}
}

func sshOptionValuesFromArgv(args []string, key string) []string {
	var values []string
	for i := 0; i < len(args); i++ {
		tok := args[i]
		if tok == "-o" {
			if i+1 >= len(args) {
				break
			}
			i++
			opt := args[i]
			k, v, ok := strings.Cut(opt, "=")
			if ok && strings.EqualFold(k, key) {
				values = append(values, v)
				continue
			}
			if strings.EqualFold(opt, key) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				values = append(values, args[i])
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

func shellArgv(t *testing.T, cmdline string) []string {
	t.Helper()
	cmd := exec.Command("sh", "-c", `eval "set -- $GIT_SSH_COMMAND"; printf '%s\0' "$@"`, "sh")
	cmd.Env = []string{"GIT_SSH_COMMAND=" + cmdline}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("shell-parse %q: %v", cmdline, err)
	}
	if len(out) == 0 {
		return nil
	}
	parts := strings.Split(string(out), "\x00")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func identityFlagValue(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
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
