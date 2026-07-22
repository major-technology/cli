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
