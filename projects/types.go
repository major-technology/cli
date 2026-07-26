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
	Connectors   []AgentConnectorRef
	Apps         []AgentAppRef
	Skills       []AgentSkillRef
	Dir          string
}

// AgentToolPermission is one agent-scoped connector tool decision. Decision
// mirrors the platform enum (always_allow | ask | always_deny); tools not
// listed get no stored row and fall back to runtime defaults.
type AgentToolPermission struct {
	Tool     string `json:"tool"`
	Decision string `json:"decision"`
}

// AgentConnectorRef associates an agent with a resource binding slot, plus
// optional per-tool permissions. Slot is resolved against
// bindings.json slots.resources at compile time; the id is resolved at deploy.
type AgentConnectorRef struct {
	Slot        string                `json:"slot"`
	Permissions []AgentToolPermission `json:"permissions,omitempty"`
}

// AgentEndpointPermission is one agent-scoped app route decision.
type AgentEndpointPermission struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Decision string `json:"decision"`
}

// AgentAppRef associates an agent with an application binding slot (the
// agent-calls-app-routes scope), plus optional per-endpoint permissions.
type AgentAppRef struct {
	Slot        string                    `json:"slot"`
	Permissions []AgentEndpointPermission `json:"permissions,omitempty"`
}

// AgentSkillRef is one skill attachment: exactly one of Slug (a project-local
// skill compiled in the same run, authored as a bare string) or Slot (a
// platform skill via bindings.json slots.skills) is set.
type AgentSkillRef struct {
	Slug string `json:"slug,omitempty"`
	Slot string `json:"slot,omitempty"`
}

// ResourceBinding is one bindings.json slots.resources entry. Type is the
// connector subtype, carried for authoring clarity and parity with the shared
// manifest shape; compile does not verify it against the platform.
type ResourceBinding struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// IDBinding is one bindings.json slots.applications / slots.skills entry.
type IDBinding struct {
	ID string `json:"id"`
}

// BindingSlots is the normalized slot manifest from bindings.json. Empty
// kinds are omitted so a project that binds only resources emits only that
// key.
type BindingSlots struct {
	Resources    map[string]ResourceBinding `json:"resources,omitempty"`
	Applications map[string]IDBinding       `json:"applications,omitempty"`
	Skills       map[string]IDBinding       `json:"skills,omitempty"`
}

// SkillDefinition is one skill directory discovered under <srcDir>/skills.
// Load only records its slug and location; Compile reads and validates the
// full bundle (skills.go).
type SkillDefinition struct {
	Slug string
	Dir  string // skill directory relative to the project root, like "src/skills/pdf-tools"
}

// LoadedProject is the fully parsed and validated project directory.
// Bindings is nil when the project has no bindings.json.
type LoadedProject struct {
	Definition ProjectDefinition
	Agents     []AgentDefinition
	Skills     []SkillDefinition
	Bindings   *BindingSlots
}

// CompiledProject is the project block of the compiled config.
type CompiledProject struct {
	Name string `json:"name"`
}

// CompiledAgent is one agent entry of the compiled config.
type CompiledAgent struct {
	Slug         string              `json:"slug"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Model        string              `json:"model,omitempty"`
	SystemPrompt string              `json:"systemPrompt"`
	Env          map[string]*string  `json:"env,omitempty"`
	Connectors   []AgentConnectorRef `json:"connectors,omitempty"`
	Apps         []AgentAppRef       `json:"apps,omitempty"`
	Skills       []AgentSkillRef     `json:"skills,omitempty"`
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
// Agents, Skills, and Bindings are optional: an empty project omits all three
// keys rather than emitting "[]" / "{}". Bindings carries the project's slot
// manifest verbatim - compile validates every agent reference against it,
// deploy resolves the slots to ids.
type CompiledConfig struct {
	ConfigVersion int             `json:"configVersion"`
	Project       CompiledProject `json:"project"`
	Agents        []CompiledAgent `json:"agents,omitempty"`
	Skills        []CompiledSkill `json:"skills,omitempty"`
	Bindings      *BindingSlots   `json:"bindings,omitempty"`
}
