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
// marshals map keys in sorted order.
func Compile(dir string) (*CompileResult, []Issue) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, []Issue{{File: "project.json", Message: "cannot resolve project directory: " + err.Error()}}
	}
	dir = absDir

	loaded, issues := Load(dir)
	if len(issues) > 0 {
		return nil, issues
	}

	agents := make([]CompiledAgent, 0, len(loaded.Agents))

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
		})
	}

	if len(issues) > 0 {
		return nil, issues
	}

	config := CompiledConfig{
		ConfigVersion: 1,
		Project:       CompiledProject{Name: loaded.Definition.Name},
		Agents:        agents,
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
