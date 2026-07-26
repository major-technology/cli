package projects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	var skills []CompiledSkill
	skillSlugs := map[string]bool{}

	for _, s := range loaded.Skills {
		compiled, skillIssues := readSkillBundle(dir, s, skillBundlesDir)
		if len(skillIssues) > 0 {
			issues = append(issues, skillIssues...)
			continue
		}

		skills = append(skills, *compiled)
		skillSlugs[compiled.Slug] = true
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

		agentFile := filepath.Join(a.Dir, "agent.json")
		unresolvedSkill := false

		for idx, skillSlug := range a.Skills {
			if !skillSlugs[skillSlug] {
				issues = append(issues, Issue{
					File:    agentFile,
					Path:    fmt.Sprintf("/skills/%d", idx),
					Message: fmt.Sprintf("agent %q references unknown skill %q", a.Slug, skillSlug),
				})
				unresolvedSkill = true
			}
		}

		if unresolvedSkill {
			continue
		}

		agents = append(agents, CompiledAgent{
			Slug:         a.Slug,
			Name:         a.Name,
			Description:  a.Description,
			Model:        a.Model,
			SystemPrompt: prompt,
			Env:          a.Env,
			Skills:       a.Skills,
		})
	}

	if len(issues) > 0 {
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
		return nil, []Issue{{File: "project.json", Message: "internal error serializing compiled config: " + err.Error()}}
	}

	sum := sha256.Sum256(configJSON)

	return &CompileResult{
		Config:     config,
		ConfigJSON: configJSON,
		Hash:       hex.EncodeToString(sum[:]),
	}, nil
}

// Validate runs the full compile pipeline and reports issues only.
func Validate(dir string) []Issue {
	_, issues := Compile(dir)
	return issues
}
