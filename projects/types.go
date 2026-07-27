package projects

// Severity classifies an Issue as blocking or advisory. The zero value is
// SeverityError, so the many existing Issue{} literals across the package
// need no changes to keep meaning "this fails validation/compile".
type Severity string

const (
	SeverityError   Severity = ""
	SeverityWarning Severity = "warning"
)

// Issue is a single validation problem tied to a file and a location within it.
type Issue struct {
	File     string   `json:"file"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity,omitempty"`
}

// IsWarning reports whether the issue is advisory rather than blocking.
func (i Issue) IsWarning() bool {
	return i.Severity == SeverityWarning
}

// PartitionIssues splits issues into blocking errors and advisory warnings,
// preserving order within each group. Callers that only care whether a run
// should fail can check len(errors) > 0.
func PartitionIssues(issues []Issue) (errors, warnings []Issue) {
	for _, issue := range issues {
		if issue.IsWarning() {
			warnings = append(warnings, issue)
		} else {
			errors = append(errors, issue)
		}
	}
	return errors, warnings
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

// SkillDefinition is one skill directory discovered under <srcDir>/skills.
// Load only records its slug and location; Compile reads and validates the
// full bundle (skills.go).
type SkillDefinition struct {
	Slug string
	Dir  string // skill directory relative to the project root, like "src/skills/pdf-tools"
}

// LoadedProject is the fully parsed and validated project directory.
type LoadedProject struct {
	Definition ProjectDefinition
	Agents     []AgentDefinition
	Skills     []SkillDefinition
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

// CompiledSkillBundle locates a skill's zipped bundle in S3. The CLI never
// sets this - it always emits Bundle nil. The mono-owned compile job uploads
// the bundle (built from the CLI's --skill-bundles-dir output, or its own
// clone of the same tree) and injects this field before submitting.
type CompiledSkillBundle struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// CompiledSkill is one skill entry of the compiled config. It never inlines
// file contents: TreeHash is the git tree object hash of the skill directory
// at HEAD (skills.go), and the platform publishes skill versions from the S3
// bundle located by Bundle once the compile job injects it.
type CompiledSkill struct {
	Slug     string               `json:"slug"`
	TreeHash string               `json:"treeHash"`
	Bundle   *CompiledSkillBundle `json:"bundle,omitempty"`
}

// CompiledConfig is the canonical compile output. The platform stores this
// JSON on project_versions.compiled_config and deploys read it verbatim.
// Agents and Skills are optional: an empty project omits both keys rather
// than emitting "[]".
type CompiledConfig struct {
	ConfigVersion int             `json:"configVersion"`
	Project       CompiledProject `json:"project"`
	Agents        []CompiledAgent `json:"agents,omitempty"`
	Skills        []CompiledSkill `json:"skills,omitempty"`
}
