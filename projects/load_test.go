package projects

import (
	"strings"
	"testing"
)

func findIssue(issues []Issue, substr string) bool {
	for _, i := range issues {
		if strings.Contains(i.Message, substr) {
			return true
		}
	}
	return false
}

func TestLoadValidProject(t *testing.T) {
	p, issues := Load("testdata/valid")
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}
	if p.Definition.Name != "Valid Project" {
		t.Fatalf("wrong project name: %q", p.Definition.Name)
	}
	if len(p.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(p.Agents))
	}
	// Agents sorted by slug: triage, writer
	if p.Agents[0].Slug != "triage" || p.Agents[1].Slug != "writer" {
		t.Fatalf("agents not sorted by slug: %s, %s", p.Agents[0].Slug, p.Agents[1].Slug)
	}
	if p.Agents[1].SystemPrompt != "You write release notes." {
		t.Fatalf("inline prompt not carried: %q", p.Agents[1].SystemPrompt)
	}
	if v, ok := p.Agents[0].Env["LINEAR_API_KEY"]; !ok || v != nil {
		t.Fatalf("null env value not preserved: %v %v", v, ok)
	}
}

func TestLoadReservedFieldRejected(t *testing.T) {
	p, issues := Load("testdata/reserved-fields")
	if p != nil {
		t.Fatal("expected nil project on issues")
	}
	if !findIssue(issues, "reserved for a future version") {
		t.Fatalf("expected reserved-field message, got: %+v", issues)
	}
	if !findIssue(issues, "schedules") {
		t.Fatalf("expected message to name the field, got: %+v", issues)
	}
}

func TestLoadMalformedJSON(t *testing.T) {
	p, issues := Load("testdata/malformed")
	if p != nil {
		t.Fatal("expected nil project on issues")
	}
	if len(issues) == 0 || !strings.Contains(issues[0].File, "agent.json") {
		t.Fatalf("expected issue on agent.json, got: %+v", issues)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	_, issues := Load("testdata/missing-required")
	if !findIssue(issues, "name") && !findIssue(issues, "systemPrompt") {
		t.Fatalf("expected schema errors naming missing fields, got: %+v", issues)
	}
}

func TestLoadDuplicateSlugs(t *testing.T) {
	_, issues := Load("testdata/duplicate-slugs")
	if !findIssue(issues, "duplicate") {
		t.Fatalf("expected duplicate slug issue, got: %+v", issues)
	}
}

func TestLoadMissingProjectJSON(t *testing.T) {
	_, issues := Load(t.TempDir())
	if !findIssue(issues, "project.json") {
		t.Fatalf("expected missing project.json issue, got: %+v", issues)
	}
}
