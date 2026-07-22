package projects

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// schemaMessagePrinter renders ErrorKind values (e.g. "missing property 'name'")
// into readable prose. ErrorKind has no String()/Error() method - only
// LocalizedString(*message.Printer) - so every render site needs a printer.
var schemaMessagePrinter = message.NewPrinter(language.English)

// reservedAgentFields are agent.json keys claimed by future versions. They are
// rejected with a dedicated message so v1 can introduce them without silent
// behavior changes on old CLIs.
var reservedAgentFields = []string{
	"schedules", "connectors", "apps", "skills", "toolPermissions", "tools", "hooks",
}

// rawSystemPrompt accepts either an inline string or {"file": "./x.md"}.
type rawSystemPrompt struct {
	Inline string
	File   string
}

func (s *rawSystemPrompt) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		s.Inline = str
		return nil
	}

	var obj struct {
		File string `json:"file"`
	}

	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}

	s.File = obj.File
	return nil
}

type rawAgent struct {
	Slug         string             `json:"slug"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Model        string             `json:"model"`
	SystemPrompt rawSystemPrompt    `json:"systemPrompt"`
	Env          map[string]*string `json:"env"`
}

type rawProject struct {
	Name   string `json:"name"`
	SrcDir string `json:"srcDir"`
}

// validateAgainst runs a compiled schema over raw JSON bytes and converts
// failures into Issues attributed to file.
func validateAgainst(schema *jsonschema.Schema, raw []byte, file string) []Issue {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(raw)))
	if err != nil {
		return []Issue{{File: file, Message: "invalid JSON: " + err.Error()}}
	}

	if err := schema.Validate(doc); err != nil {
		var ve *jsonschema.ValidationError
		if ok := errorsAs(err, &ve); ok {
			var issues []Issue
			for _, cause := range flattenValidationError(ve) {
				issues = append(issues, Issue{
					File:    file,
					Path:    cause.path,
					Message: cause.message,
				})
			}
			return issues
		}
		return []Issue{{File: file, Message: err.Error()}}
	}

	return nil
}

type schemaCause struct {
	path    string
	message string
}

// flattenValidationError walks a jsonschema ValidationError tree into leaves.
func flattenValidationError(ve *jsonschema.ValidationError) []schemaCause {
	if len(ve.Causes) == 0 {
		return []schemaCause{{
			path:    "/" + strings.Join(ve.InstanceLocation, "/"),
			message: ve.ErrorKind.LocalizedString(schemaMessagePrinter),
		}}
	}

	var out []schemaCause
	for _, c := range ve.Causes {
		out = append(out, flattenValidationError(c)...)
	}

	return out
}

// errorsAs is a tiny wrapper so load.go does not import errors twice under
// different names elsewhere in the package.
func errorsAs(err error, target **jsonschema.ValidationError) bool {
	if v, ok := err.(*jsonschema.ValidationError); ok {
		*target = v
		return true
	}
	return false
}

// checkReservedFields reports any reserved key present in the raw agent JSON.
func checkReservedFields(raw []byte, file string) []Issue {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return nil // malformed JSON is reported by validateAgainst
	}

	var issues []Issue
	for _, field := range reservedAgentFields {
		if _, present := keys[field]; present {
			issues = append(issues, Issue{
				File:    file,
				Path:    "/" + field,
				Message: fmt.Sprintf("%q is reserved for a future version and is not supported in v0", field),
			})
		}
	}

	return issues
}

// stripReservedFields returns raw with any reserved agent.json keys removed.
// The agent schema has additionalProperties:false and does not list the
// reserved keys, so validating the untouched document would also surface a
// generic "additional property" schema error alongside checkReservedFields'
// dedicated message. Stripping them first means the reserved-field message is
// the only one reported for those keys; genuine schema errors (missing
// required fields, wrong types, unrecognized non-reserved fields, ...) still
// surface normally.
func stripReservedFields(raw []byte) []byte {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return raw // malformed JSON is reported by validateAgainst
	}

	changed := false
	for _, field := range reservedAgentFields {
		if _, present := keys[field]; present {
			delete(keys, field)
			changed = true
		}
	}

	if !changed {
		return raw
	}

	cleaned, err := json.Marshal(keys)
	if err != nil {
		return raw
	}

	return cleaned
}

// Load parses and validates a project directory. It returns the loaded
// project, or nil plus one or more issues. For {"file": ...} prompts the
// SystemPrompt is a "file:<ref>" placeholder that Compile (compile.go)
// replaces with the inlined contents.
func Load(dir string) (*LoadedProject, []Issue) {
	projectPath := filepath.Join(dir, "project.json")

	rawProjectBytes, err := os.ReadFile(projectPath)
	if err != nil {
		return nil, []Issue{{File: "project.json", Message: "project.json not found - is this a major project directory?"}}
	}

	var issues []Issue
	issues = append(issues, validateAgainst(ProjectSchema(), rawProjectBytes, "project.json")...)

	var proj rawProject
	if len(issues) == 0 {
		if err := json.Unmarshal(rawProjectBytes, &proj); err != nil {
			issues = append(issues, Issue{File: "project.json", Message: "invalid JSON: " + err.Error()})
		}
	}

	if len(issues) > 0 {
		return nil, issues
	}

	srcDir := proj.SrcDir
	if srcDir == "" {
		srcDir = "src/"
	}

	agentsDir := filepath.Join(dir, srcDir, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		// No agents directory means an empty (but valid) project.
		return &LoadedProject{Definition: ProjectDefinition{Name: proj.Name, SrcDir: srcDir}}, nil
	}

	var agents []AgentDefinition
	seenSlugs := map[string]string{} // slug -> file that declared it

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		agentFile := filepath.Join(srcDir, "agents", entry.Name(), "agent.json")
		agentPath := filepath.Join(dir, agentFile)

		rawBytes, err := os.ReadFile(agentPath)
		if err != nil {
			continue // directories without agent.json are ignored
		}

		// Reserved-field check runs first, and schema validation runs against
		// a copy with reserved keys stripped, so a reserved field is reported
		// once with the dedicated message instead of also tripping the
		// schema's generic additionalProperties error.
		fileIssues := checkReservedFields(rawBytes, agentFile)
		fileIssues = append(fileIssues, validateAgainst(AgentSchema(), stripReservedFields(rawBytes), agentFile)...)

		if len(fileIssues) > 0 {
			issues = append(issues, fileIssues...)
			continue
		}

		var ra rawAgent
		if err := json.Unmarshal(rawBytes, &ra); err != nil {
			issues = append(issues, Issue{File: agentFile, Message: "invalid JSON: " + err.Error()})
			continue
		}

		if prev, dup := seenSlugs[ra.Slug]; dup {
			issues = append(issues, Issue{
				File:    agentFile,
				Path:    "/slug",
				Message: fmt.Sprintf("duplicate slug %q (already declared in %s)", ra.Slug, prev),
			})
			continue
		}
		seenSlugs[ra.Slug] = agentFile

		systemPrompt := ra.SystemPrompt.Inline
		if ra.SystemPrompt.File != "" {
			systemPrompt = "file:" + ra.SystemPrompt.File
		}

		agents = append(agents, AgentDefinition{
			Slug:         ra.Slug,
			Name:         ra.Name,
			Description:  ra.Description,
			Model:        ra.Model,
			SystemPrompt: systemPrompt,
			Env:          ra.Env,
			Dir:          filepath.Join(srcDir, "agents", entry.Name()),
		})
	}

	if len(issues) > 0 {
		return nil, issues
	}

	sort.Slice(agents, func(i, j int) bool { return agents[i].Slug < agents[j].Slug })

	return &LoadedProject{
		Definition: ProjectDefinition{Name: proj.Name, SrcDir: srcDir},
		Agents:     agents,
	}, nil
}
