package project

import (
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
)

func TestRenderPlanShowsAllBuckets(t *testing.T) {
	plan := &api.GetProjectDeployPlanResponse{
		Creates:   []string{"new-agent"},
		Updates:   []string{"triage"},
		Unchanged: []string{"writer"},
		Deletes:   []string{"old-agent"},
	}

	out := renderPlan(plan)

	for _, want := range []string{"new-agent", "triage", "writer", "old-agent", "create", "update", "unchanged", "delete"} {
		if !strings.Contains(strings.ToLower(out), want) {
			t.Fatalf("plan output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderPlanEmptyProject(t *testing.T) {
	out := renderPlan(&api.GetProjectDeployPlanResponse{})
	if !strings.Contains(strings.ToLower(out), "no changes") {
		t.Fatalf("expected 'no changes' for empty plan, got: %s", out)
	}
}

func TestRenderPlanShowsWarningsWithNoOtherChanges(t *testing.T) {
	plan := &api.GetProjectDeployPlanResponse{
		Warnings: []string{"env var FOO declared null but no value set"},
	}

	out := renderPlan(plan)

	if strings.Contains(strings.ToLower(out), "no changes") {
		t.Fatalf("expected warnings to suppress 'no changes', got: %s", out)
	}
	if !strings.Contains(strings.ToLower(out), "warning") {
		t.Fatalf("expected a warnings section, got: %s", out)
	}
	if !strings.Contains(out, "FOO") {
		t.Fatalf("expected warning text in output, got: %s", out)
	}
}

func TestRenderPlanNoWarningsSectionWhenEmpty(t *testing.T) {
	plan := &api.GetProjectDeployPlanResponse{
		Creates: []string{"new-agent"},
	}

	out := renderPlan(plan)

	if strings.Contains(strings.ToLower(out), "warning") {
		t.Fatalf("expected no warnings section when Warnings is empty, got: %s", out)
	}
}

func TestResolveVersionDefaultsToNewestCompiled(t *testing.T) {
	versions := []api.ProjectVersionItem{
		{ID: "v-2", CommitHash: "bbbb222222bbbb222222bbbb222222bbbb22222", CompileStatus: "compiled", CreatedAt: "2026-07-21T00:00:00Z"},
		{ID: "v-1", CommitHash: "aaaa111111aaaa111111aaaa111111aaaa11111", CompileStatus: "compiled", CreatedAt: "2026-07-20T00:00:00Z"},
	}

	got, err := resolveVersion(versions, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "v-2" {
		t.Fatalf("expected newest (first-listed) compiled version v-2, got %+v", got)
	}
}

func TestResolveVersionSkipsFailedVersionsForDefault(t *testing.T) {
	versions := []api.ProjectVersionItem{
		{ID: "v-3", CommitHash: "cccc333333cccc333333cccc333333cccc33333", CompileStatus: "failed", CompileError: "boom", CreatedAt: "2026-07-22T00:00:00Z"},
		{ID: "v-2", CommitHash: "bbbb222222bbbb222222bbbb222222bbbb22222", CompileStatus: "compiled", CreatedAt: "2026-07-21T00:00:00Z"},
	}

	got, err := resolveVersion(versions, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "v-2" {
		t.Fatalf("expected default to skip failed v-3 and pick compiled v-2, got %+v", got)
	}
}

func TestResolveVersionExactMatch(t *testing.T) {
	versions := []api.ProjectVersionItem{
		{ID: "v-1", CommitHash: "aaaa111111aaaa111111aaaa111111aaaa11111", CompileStatus: "compiled"},
		{ID: "v-2", CommitHash: "aaaa111111aaaa111111aaaa111111aaaa11112", CompileStatus: "compiled"},
	}

	got, err := resolveVersion(versions, "aaaa111111aaaa111111aaaa111111aaaa11111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "v-1" {
		t.Fatalf("expected exact match v-1, got %+v", got)
	}
}

func TestResolveVersionUniquePrefix(t *testing.T) {
	versions := []api.ProjectVersionItem{
		{ID: "v-1", CommitHash: "abc123def456", CompileStatus: "compiled"},
		{ID: "v-2", CommitHash: "def456abc123", CompileStatus: "compiled"},
	}

	got, err := resolveVersion(versions, "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "v-1" {
		t.Fatalf("expected unique prefix match v-1, got %+v", got)
	}
}

func TestResolveVersionAmbiguousPrefix(t *testing.T) {
	versions := []api.ProjectVersionItem{
		{ID: "v-1", CommitHash: "abc123def456", CompileStatus: "compiled", CreatedAt: "2026-07-20T00:00:00Z"},
		{ID: "v-2", CommitHash: "abc123abcdef", CompileStatus: "compiled", CreatedAt: "2026-07-21T00:00:00Z"},
	}

	_, err := resolveVersion(versions, "abc123")
	if err == nil {
		t.Fatalf("expected ambiguous prefix error")
	}

	msg := err.Error()
	for _, want := range []string{"abc123def456", "abc123abcdef", "2026-07-20T00:00:00Z", "2026-07-21T00:00:00Z"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error to mention %q, got: %s", want, msg)
		}
	}
}

func TestResolveVersionNoMatch(t *testing.T) {
	versions := []api.ProjectVersionItem{
		{ID: "v-1", CommitHash: "abc123def456", CompileStatus: "compiled"},
	}

	_, err := resolveVersion(versions, "zzz")
	if err == nil {
		t.Fatalf("expected error for no match, got version")
	}
}
