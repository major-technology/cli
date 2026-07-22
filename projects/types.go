package projects

// Issue is a single validation problem tied to a file and a location within it.
type Issue struct {
	File    string `json:"file"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// ProjectDefinition is the parsed project.json.
type ProjectDefinition struct {
	Name   string
	SrcDir string
}

// AgentDefinition is one parsed agent.json with its system prompt inlined.
// Dir is the agent's directory relative to the project root (like
// "src/agents/triage"), used to resolve relative prompt-file references.
type AgentDefinition struct {
	Slug         string
	Name         string
	Description  string
	Model        string
	SystemPrompt string
	Env          map[string]*string
	Dir          string
}

// LoadedProject is the fully parsed and validated project directory.
type LoadedProject struct {
	Definition ProjectDefinition
	Agents     []AgentDefinition
}

// CompiledProject is the project block of the compiled config.
type CompiledProject struct {
	Name string `json:"name"`
}

// CompiledAgent is one agent entry of the compiled config.
type CompiledAgent struct {
	Slug         string             `json:"slug"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Model        string             `json:"model,omitempty"`
	SystemPrompt string             `json:"systemPrompt"`
	Env          map[string]*string `json:"env,omitempty"`
}

// CompiledConfig is the canonical compile output. The platform stores this
// JSON on project_versions.compiled_config and deploys read it verbatim.
type CompiledConfig struct {
	ConfigVersion int             `json:"configVersion"`
	Project       CompiledProject `json:"project"`
	Agents        []CompiledAgent `json:"agents"`
}
