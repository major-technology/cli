package projects

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testResourceID    = "11111111-1111-4111-8111-111111111111"
	testApplicationID = "22222222-2222-4222-8222-222222222222"
	testSkillID       = "33333333-3333-4333-8333-333333333333"
)

// writeBindings writes a bindings.json body into the project root.
func writeBindings(t *testing.T, dir, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, BindingsFileName), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// writeAgent writes an agent.json under <dir>/src/agents/<slug>/.
func writeAgent(t *testing.T, dir, slug, body string) {
	t.Helper()

	agentDir := filepath.Join(dir, "src", "agents", slug)
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent.json"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// writeSkill writes a minimal valid skill under <dir>/src/skills/<slug>/.
func writeSkill(t *testing.T, dir, slug string) {
	t.Helper()

	skillDir := filepath.Join(dir, "src", "skills", slug)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + slug + "\ndescription: A test skill.\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

const fullBindings = `{
	"$schema": "https://schemas.major.tech/bindings.json",
	"slots": {
		"resources": { "prod-db": { "type": "postgresql", "id": "` + testResourceID + `" } },
		"applications": { "crm": { "id": "` + testApplicationID + `" } },
		"skills": { "reporting": { "id": "` + testSkillID + `" } }
	}
}`

const fullAgent = `{
	"slug": "triage",
	"name": "Triage",
	"systemPrompt": "You triage.",
	"connectors": [{ "slot": "prod-db", "permissions": [{ "tool": "postgresql_psql", "decision": "always_allow" }] }],
	"apps": [{ "slot": "crm", "permissions": [{ "method": "POST", "path": "/api/orders", "decision": "ask" }] }],
	"skills": ["pdf-tools", { "slot": "reporting" }]
}`

// TestCompileFullAgentWithBindings is the happy path: all three facets on one
// agent, every reference resolving through bindings.json or a project-local
// skill directory.
func TestCompileFullAgentWithBindings(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeBindings(t, dir, fullBindings)
	writeSkill(t, dir, "pdf-tools")
	writeAgent(t, dir, "triage", fullAgent)
	initGitRepo(t, dir)

	res, issues := Compile(dir)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	agent := res.Config.Agents[0]
	if len(agent.Connectors) != 1 || agent.Connectors[0].ResourceID != testResourceID {
		t.Fatalf("connectors = %+v", agent.Connectors)
	}
	if p := agent.Connectors[0].Permissions; len(p) != 1 || p[0].Tool != "postgresql_psql" || p[0].Decision != "always_allow" {
		t.Fatalf("connector permissions = %+v", p)
	}
	if len(agent.Apps) != 1 || agent.Apps[0].ApplicationID != testApplicationID {
		t.Fatalf("apps = %+v", agent.Apps)
	}
	if p := agent.Apps[0].Permissions; len(p) != 1 || p[0].Method != "POST" || p[0].Path != "/api/orders" || p[0].Decision != "ask" {
		t.Fatalf("app permissions = %+v", p)
	}

	// A bare string stays a slug (the local skill has no id until this deploy
	// publishes it); a slot resolves to the bound platform skill id.
	want := []CompiledSkillRef{{Slug: "pdf-tools"}, {SkillID: testSkillID}}
	if len(agent.Skills) != len(want) {
		t.Fatalf("skills = %+v", agent.Skills)
	}
	for i := range want {
		if agent.Skills[i] != want[i] {
			t.Fatalf("skills[%d] = %+v, want %+v", i, agent.Skills[i], want[i])
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(res.ConfigJSON, &raw); err != nil {
		t.Fatalf("compiled JSON does not parse: %v", err)
	}
	if _, ok := raw["bindings"]; ok {
		t.Fatalf("the slot manifest must not reach the compiled config: %s", res.ConfigJSON)
	}
	if !strings.Contains(string(res.ConfigJSON), `"skills":[{"slug":"pdf-tools"},{"skillId":"`+testSkillID+`"}]`) {
		t.Fatalf("skills not resolved as expected: %s", res.ConfigJSON)
	}

	// No slot name survives compilation anywhere in the output.
	for _, slot := range []string{"prod-db", "crm", "reporting"} {
		if strings.Contains(string(res.ConfigJSON), slot) {
			t.Fatalf("slot %q leaked into the compiled config: %s", slot, res.ConfigJSON)
		}
	}
}

// TestCompileNeverEmitsBindingsKey checks that a project WITH bindings.json
// still emits no top-level bindings key: the manifest is compile-time input
// only, with no downstream consumer.
func TestCompileNeverEmitsBindingsKey(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeBindings(t, dir, fullBindings)

	res, issues := Compile(dir)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	if strings.Contains(string(res.ConfigJSON), `"bindings"`) {
		t.Fatalf("expected no bindings key: %s", res.ConfigJSON)
	}
	for _, id := range []string{testResourceID, testApplicationID, testSkillID} {
		if strings.Contains(string(res.ConfigJSON), id) {
			t.Fatalf("unreferenced binding id %q leaked into the compiled config: %s", id, res.ConfigJSON)
		}
	}
}

// TestCompileWithoutBindingsFileOmitsBindingsKey pins backward compatibility:
// a project that predates bindings.json compiles to exactly the bytes it did
// before, with no bindings key and no facet keys.
func TestCompileWithoutBindingsFileOmitsBindingsKey(t *testing.T) {
	res, issues := Compile("testdata/valid")
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	for _, key := range []string{`"bindings"`, `"connectors"`, `"apps"`, `"skills"`} {
		if strings.Contains(string(res.ConfigJSON), key) {
			t.Fatalf("expected no %s key for an old project, got: %s", key, res.ConfigJSON)
		}
	}
}

func TestCompileUnknownConnectorSlotRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeBindings(t, dir, fullBindings)
	writeAgent(t, dir, "triage", `{"slug":"triage","name":"T","systemPrompt":"x","connectors":[{"slot":"staging-db"}]}`)

	_, issues := Compile(dir)
	if !findIssue(issues, `unknown binding slot "staging-db"`) || !findIssue(issues, "slots.resources") {
		t.Fatalf("expected an unknown resource-slot issue, got: %+v", issues)
	}
}

func TestCompileUnknownAppSlotRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeBindings(t, dir, fullBindings)
	writeAgent(t, dir, "triage", `{"slug":"triage","name":"T","systemPrompt":"x","apps":[{"slot":"billing"}]}`)

	_, issues := Compile(dir)
	if !findIssue(issues, `unknown binding slot "billing"`) || !findIssue(issues, "slots.applications") {
		t.Fatalf("expected an unknown application-slot issue, got: %+v", issues)
	}
}

func TestCompileUnknownSkillSlotRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeBindings(t, dir, fullBindings)
	writeAgent(t, dir, "triage", `{"slug":"triage","name":"T","systemPrompt":"x","skills":[{"slot":"forecasting"}]}`)

	_, issues := Compile(dir)
	if !findIssue(issues, `unknown binding slot "forecasting"`) || !findIssue(issues, "slots.skills") {
		t.Fatalf("expected an unknown skill-slot issue, got: %+v", issues)
	}
}

func TestCompileUnknownProjectSkillRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeSkill(t, dir, "pdf-tools")
	writeAgent(t, dir, "triage", `{"slug":"triage","name":"T","systemPrompt":"x","skills":["ocr-tools"]}`)
	initGitRepo(t, dir)

	_, issues := Compile(dir)
	if !findIssue(issues, `skill "ocr-tools" is not a skill in this project`) {
		t.Fatalf("expected an unresolved project-skill issue, got: %+v", issues)
	}
}

// TestCompileSlotReferenceWithoutBindingsFile checks the no-manifest case gets
// its own message telling the author to add bindings.json, not a confusing
// "unknown slot" against an empty manifest.
func TestCompileSlotReferenceWithoutBindingsFile(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeAgent(t, dir, "triage", `{"slug":"triage","name":"T","systemPrompt":"x","connectors":[{"slot":"prod-db"}]}`)

	_, issues := Compile(dir)
	if !findIssue(issues, "this project has no bindings.json") {
		t.Fatalf("expected a missing-bindings.json issue, got: %+v", issues)
	}
}

func TestCompileInvalidBindingsRejected(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"malformed JSON", `{"slots":`, "invalid JSON"},
		{"unknown top-level key", `{"slots":{},"extra":1}`, "extra"},
		{"unknown slot kind", `{"slots":{"widgets":{}}}`, "widgets"},
		{"non-uuid id", `{"slots":{"applications":{"crm":{"id":"not-a-uuid"}}}}`, "pattern"},
		{"empty resource type", `{"slots":{"resources":{"db":{"type":"","id":"` + testResourceID + `"}}}}`, "minLength"},
		{"bad slot charset", `{"slots":{"applications":{"Bad_Slot":{"id":"` + testApplicationID + `"}}}}`, "pattern"},
		{"unknown entry key", `{"slots":{"applications":{"crm":{"id":"` + testApplicationID + `","name":"x"}}}}`, "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixtureProject(t, dir)
			writeBindings(t, dir, tt.body)

			_, issues := Compile(dir)
			if len(issues) == 0 {
				t.Fatal("expected an issue")
			}
			if issues[0].File != BindingsFileName {
				t.Fatalf("issue not attributed to %s: %+v", BindingsFileName, issues)
			}
			if !findIssue(issues, tt.want) {
				t.Fatalf("expected an issue mentioning %q, got: %+v", tt.want, issues)
			}
		})
	}
}

func TestCompileDuplicateFacetEntriesRejected(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		want  string
	}{
		{
			name:  "duplicate connector slot",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","connectors":[{"slot":"prod-db"},{"slot":"prod-db"}]}`,
			want:  `duplicate connector slot "prod-db"`,
		},
		{
			name:  "duplicate app slot",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","apps":[{"slot":"crm"},{"slot":"crm"}]}`,
			want:  `duplicate app slot "crm"`,
		},
		{
			name:  "duplicate connector tool",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","connectors":[{"slot":"prod-db","permissions":[{"tool":"t","decision":"ask"},{"tool":"t","decision":"always_deny"}]}]}`,
			want:  `duplicate permission for tool "t"`,
		},
		{
			name:  "duplicate app endpoint",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","apps":[{"slot":"crm","permissions":[{"method":"GET","path":"/x","decision":"ask"},{"method":"GET","path":"/x","decision":"always_allow"}]}]}`,
			want:  "duplicate permission for GET /x",
		},
		{
			name:  "duplicate skill slot",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","skills":[{"slot":"reporting"},{"slot":"reporting"}]}`,
			want:  `duplicate reference to skill slot "reporting"`,
		},
		{
			name:  "duplicate project skill",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","skills":["pdf-tools","pdf-tools"]}`,
			want:  `duplicate reference to project skill "pdf-tools"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixtureProject(t, dir)
			writeBindings(t, dir, fullBindings)
			writeSkill(t, dir, "pdf-tools")
			writeAgent(t, dir, "a", tt.agent)
			initGitRepo(t, dir)

			_, issues := Compile(dir)
			if !findIssue(issues, tt.want) {
				t.Fatalf("expected an issue mentioning %q, got: %+v", tt.want, issues)
			}
		})
	}
}

func TestCompileInvalidAgentFacetShapeRejected(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		want  string
	}{
		{
			name:  "unknown decision",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","connectors":[{"slot":"prod-db","permissions":[{"tool":"t","decision":"maybe"}]}]}`,
			want:  "must be one of",
		},
		{
			name:  "connector without slot",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","connectors":[{"permissions":[]}]}`,
			want:  "slot",
		},
		{
			name:  "app permission without path",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","apps":[{"slot":"crm","permissions":[{"method":"GET","decision":"ask"}]}]}`,
			want:  "path",
		},
		{
			name:  "unknown key on a connector",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","connectors":[{"slot":"prod-db","upstream":true}]}`,
			want:  "upstream",
		},
		{
			name:  "skill ref that is neither a string nor a slot object",
			agent: `{"slug":"a","name":"A","systemPrompt":"x","skills":[{"slug":"pdf-tools"}]}`,
			want:  "slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixtureProject(t, dir)
			writeBindings(t, dir, fullBindings)
			writeAgent(t, dir, "a", tt.agent)

			_, issues := Compile(dir)
			if !findIssue(issues, tt.want) {
				t.Fatalf("expected an issue mentioning %q, got: %+v", tt.want, issues)
			}
		})
	}
}

// TestCompileSkillSlotMayShareNameWithProjectSkill pins that a skills slot
// named after a project-local skill is NOT a conflict: compile resolves the
// slot to {"skillId"} and the local slug to {"slug"}, two structurally
// distinct shapes, so there is no merged namespace for them to collide in.
func TestCompileSkillSlotMayShareNameWithProjectSkill(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeBindings(t, dir, `{"slots":{"skills":{"pdf-tools":{"id":"`+testSkillID+`"}}}}`)
	writeSkill(t, dir, "pdf-tools")
	writeAgent(t, dir, "a", `{"slug":"a","name":"A","systemPrompt":"x","skills":["pdf-tools",{"slot":"pdf-tools"}]}`)
	initGitRepo(t, dir)

	res, issues := Compile(dir)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	want := []CompiledSkillRef{{Slug: "pdf-tools"}, {SkillID: testSkillID}}
	got := res.Config.Agents[0].Skills
	if len(got) != len(want) {
		t.Fatalf("skills = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("skills[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestLoadStillReservedFields pins that unreserving connectors/apps/skills did
// not unreserve the rest.
func TestLoadStillReservedFields(t *testing.T) {
	for _, field := range []string{"schedules", "toolPermissions", "tools", "hooks"} {
		t.Run(field, func(t *testing.T) {
			dir := t.TempDir()
			writeFixtureProject(t, dir)
			writeAgent(t, dir, "a", `{"slug":"a","name":"A","systemPrompt":"x","`+field+`":[]}`)

			_, issues := Load(dir)
			if !findIssue(issues, "reserved for a future version") || !findIssue(issues, field) {
				t.Fatalf("expected %q to stay reserved, got: %+v", field, issues)
			}
			if len(issues) != 1 {
				t.Fatalf("expected exactly one issue for %q, got: %+v", field, issues)
			}
		})
	}
}

// TestLoadFacetsNoLongerReserved is the mirror of the test above: the three
// unreserved fields must parse instead of erroring.
func TestLoadFacetsNoLongerReserved(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	writeBindings(t, dir, fullBindings)
	writeAgent(t, dir, "a", `{"slug":"a","name":"A","systemPrompt":"x","connectors":[{"slot":"prod-db"}],"apps":[{"slot":"crm"}],"skills":[{"slot":"reporting"}]}`)

	loaded, issues := Load(dir)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}
	if len(loaded.Agents[0].Connectors) != 1 || len(loaded.Agents[0].Apps) != 1 || len(loaded.Agents[0].Skills) != 1 {
		t.Fatalf("facets not parsed: %+v", loaded.Agents[0])
	}
}
