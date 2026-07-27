package projects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
)

// CompileResult is the canonical compile output plus its serialized form.
type CompileResult struct {
	Config     CompiledConfig
	ConfigJSON []byte
	Hash       string
}

// Compile loads, validates, and compiles a project directory. Returns the
// result, or nil plus issues. ConfigJSON is deterministic for identical
// inputs: struct field order is fixed, agents are sorted by slug, and Go
// marshals map keys in sorted order. Skill bundle zips are not written; use
// CompileWithSkillBundles for that.
func Compile(dir string) (*CompileResult, []Issue) {
	return compile(dir, "")
}

// CompileWithSkillBundles is Compile plus writing each skill's zip bundle to
// <skillBundlesDir>/<treeHash>.zip (cmd/project/compile.go's
// --skill-bundles-dir). The CLI itself never uploads these to S3; the
// mono-owned compile job does that and injects CompiledSkill.Bundle before
// submitting.
func CompileWithSkillBundles(dir, skillBundlesDir string) (*CompileResult, []Issue) {
	return compile(dir, skillBundlesDir)
}

func compile(dir, skillBundlesDir string) (*CompileResult, []Issue) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, []Issue{{File: "project.json", Message: "cannot resolve project directory: " + err.Error()}}
	}
	dir = absDir

	loaded, issues := Load(dir)
	if len(issues) > 0 {
		return nil, issues
	}

	var agents []CompiledAgent

	for _, a := range loaded.Agents {
		prompt := a.SystemPrompt

		if ref, isFile := strings.CutPrefix(a.SystemPrompt, "file:"); isFile {
			agentFile := filepath.Join(a.Dir, "agent.json")

			inlined, issue := readPromptFile(dir, a.Dir, ref, agentFile)
			if issue != nil {
				issues = append(issues, *issue)
				continue
			}

			prompt = inlined
		}

		agents = append(agents, CompiledAgent{
			Slug:         a.Slug,
			Name:         a.Name,
			Description:  a.Description,
			Model:        a.Model,
			SystemPrompt: prompt,
			Env:          a.Env,
			Connectors:   resolveConnectors(a.Connectors, loaded.Bindings),
			Apps:         resolveApps(a.Apps, loaded.Bindings),
			Skills:       resolveSkillRefs(a.Skills, loaded.Bindings),
		})
	}

	var skills []CompiledSkill

	for _, s := range loaded.Skills {
		compiled, skillIssues := readSkillBundle(dir, s, skillBundlesDir)
		issues = append(issues, skillIssues...)

		// compiled is nil either because of a genuine content error (caught
		// below by the errors check) or because the skill's git tree hash
		// could not be resolved - a warning-only skip (readSkillBundle), not
		// a compile failure.
		if compiled == nil {
			continue
		}

		skills = append(skills, *compiled)
	}

	if errs, _ := PartitionIssues(issues); len(errs) > 0 {
		return nil, issues
	}

	config := CompiledConfig{
		ConfigVersion: 1,
		Project:       CompiledProject{Name: loaded.Definition.Name},
		Agents:        agents,
		Skills:        skills,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, append(issues, Issue{File: "project.json", Message: "internal error serializing compiled config: " + err.Error()})
	}

	sum := sha256.Sum256(configJSON)

	return &CompileResult{
		Config:     config,
		ConfigJSON: configJSON,
		Hash:       hex.EncodeToString(sum[:]),
	}, issues // issues may be non-nil here too: warning-only issues (e.g. an uncommitted skill dir) do not block compile.
}

// Validate loads and validates a project directory: schema checks, agent and
// skill discovery, and skill file-content checks (dotfiles, size caps, path
// length, symlinks, frontmatter). It never touches git: tree-hash resolution
// only happens on the compile path (readSkillBundle), so an uncommitted or
// never-committed skill directory - the artifact that ships is the git tree
// at a commit, resolved by the platform's compile job, not the working tree -
// validates cleanly as long as its content is valid.
func Validate(dir string) []Issue {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return []Issue{{File: "project.json", Message: "cannot resolve project directory: " + err.Error()}}
	}

	_, issues := Load(absDir)
	return issues
}
