package projects

import (
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestSchemasCompile(t *testing.T) {
	tests := []struct {
		name   string
		schema func() *jsonschema.Schema
	}{
		{"project", ProjectSchema},
		{"agent", AgentSchema},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.schema() == nil {
				t.Fatalf("%s schema returned nil", tt.name)
			}
		})
	}
}

func TestAgentSchemaValidate(t *testing.T) {
	tests := []struct {
		name    string
		doc     map[string]any
		wantErr bool
	}{
		{
			name: "minimal agent accepted",
			doc: map[string]any{
				"slug":         "triage",
				"name":         "Triage",
				"systemPrompt": "You are a triage agent.",
			},
			wantErr: false,
		},
		{
			name: "unknown field rejected",
			doc: map[string]any{
				"slug":         "triage",
				"name":         "Triage",
				"systemPrompt": "You are a triage agent.",
				"bogus":        true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AgentSchema().Validate(tt.doc)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got: %v", err)
			}
		})
	}
}
