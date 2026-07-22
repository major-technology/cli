// Package projects loads, validates, and compiles a major project directory
// (plain-JSON agent definitions) into the canonical compiled config that the
// platform stores and deploys. It is the single compile implementation: the
// same code runs locally (major project validate/compile) and inside the
// webhook compile job (major project compile-and-report).
package projects

import (
	"bytes"
	"embed"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/*.schema.json
var SchemaFS embed.FS

var (
	schemaOnce    sync.Once
	projectSchema *jsonschema.Schema
	agentSchema   *jsonschema.Schema
)

// ensureSchemas compiles both embedded schemas exactly once.
func ensureSchemas() {
	schemaOnce.Do(func() {
		c := jsonschema.NewCompiler()

		for name, id := range map[string]string{
			"schemas/project.schema.json": "https://schemas.major.tech/project.json",
			"schemas/agent.schema.json":   "https://schemas.major.tech/agent.json",
		} {
			raw, err := SchemaFS.ReadFile(name)
			if err != nil {
				panic("projects: embedded schema missing: " + name)
			}

			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
			if err != nil {
				panic("projects: embedded schema is invalid JSON: " + name)
			}

			if err := c.AddResource(id, doc); err != nil {
				panic("projects: failed to add schema resource: " + err.Error())
			}
		}

		var err error

		projectSchema, err = c.Compile("https://schemas.major.tech/project.json")
		if err != nil {
			panic("projects: project schema failed to compile: " + err.Error())
		}

		agentSchema, err = c.Compile("https://schemas.major.tech/agent.json")
		if err != nil {
			panic("projects: agent schema failed to compile: " + err.Error())
		}
	})
}

// ProjectSchema returns the compiled project.json schema.
func ProjectSchema() *jsonschema.Schema {
	ensureSchemas()
	return projectSchema
}

// AgentSchema returns the compiled agent.json schema.
func AgentSchema() *jsonschema.Schema {
	ensureSchemas()
	return agentSchema
}
