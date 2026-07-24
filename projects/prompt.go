package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxPromptFileBytes caps prompt files at 200KB.
const MaxPromptFileBytes = 200 * 1024

// readPromptFile inlines a prompt file referenced from an agent.json. ref is
// the raw reference (like "./prompt.md"), agentDir the directory holding the
// agent.json (relative to projectDir), file the agent.json path for issue
// attribution. Containment: the resolved file must live inside projectDir and
// must not itself be a symlink.
func readPromptFile(projectDir, agentDir, ref, file string) (string, *Issue) {
	cleanProject := filepath.Clean(projectDir)
	candidate := filepath.Clean(filepath.Join(projectDir, agentDir, ref))

	// Lexical containment check first, before touching the filesystem: a
	// "../"-escape must be rejected as "outside the project directory" even
	// when the target does not exist, rather than being masked as "not
	// found".
	if candidate != cleanProject && !strings.HasPrefix(candidate, cleanProject+string(filepath.Separator)) {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: fmt.Sprintf("prompt file %q resolves outside the project directory", ref)}
	}

	info, err := os.Lstat(candidate)
	if err != nil {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: fmt.Sprintf("prompt file %q not found", ref)}
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: fmt.Sprintf("prompt file %q is a symlink; symlinks are not allowed", ref)}
	}

	if info.Size() > MaxPromptFileBytes {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: fmt.Sprintf("prompt file %q exceeds the 200KB limit", ref)}
	}

	// Re-check containment after resolving symlinks, so a symlinked parent
	// directory inside the project cannot smuggle the real path outside it.
	realProject, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		return "", &Issue{File: file, Message: "cannot resolve project directory: " + err.Error()}
	}

	realCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: "cannot resolve prompt file path: " + err.Error()}
	}

	if realCandidate != realProject && !strings.HasPrefix(realCandidate, realProject+string(filepath.Separator)) {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: fmt.Sprintf("prompt file %q resolves outside the project directory", ref)}
	}

	contents, err := os.ReadFile(realCandidate)
	if err != nil {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: "cannot read prompt file: " + err.Error()}
	}

	if strings.TrimSpace(string(contents)) == "" {
		return "", &Issue{File: file, Path: "/systemPrompt/file", Message: fmt.Sprintf("prompt file %q is empty", ref)}
	}

	return string(contents), nil
}
