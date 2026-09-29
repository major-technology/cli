package cmd

import (
	"net/http"
	"strings"
	"testing"
)

const relatedWorkflowID = "22222222-2222-4222-8222-222222222222"

func relatedArtifactsBody(anchorKind, anchorID string) string {
	return `{"artifacts":[` +
		`{"kind":"` + anchorKind + `","id":"` + anchorID + `","name":"Anchor","accessible":true,"isAnchor":true},` +
		`{"kind":"agent","id":"33333333-3333-4333-8333-333333333333","name":"Triage agent","accessible":false,"isAnchor":false}]}`
}

func TestRelatedArtifactsDefaultsToWorkspaceTarget(t *testing.T) {
	url := contractServer(t, map[string]http.HandlerFunc{
		"GET /artifact-family/app/" + tableAppID: func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, relatedArtifactsBody("app", tableAppID))
		},
	})

	stdout, _, err := runContractCommand(t, url, []string{"related-artifacts"}, nil, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, _ := decodeOneObject(t, stdout)["artifacts"].([]any)
	if len(artifacts) != 2 {
		t.Fatalf("want 2 artifacts, stdout=%q", stdout)
	}
}

func TestRelatedArtifactsFlagsOverrideWorkspace(t *testing.T) {
	url := contractServer(t, map[string]http.HandlerFunc{
		"GET /artifact-family/workflow/" + relatedWorkflowID: func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, relatedArtifactsBody("workflow", relatedWorkflowID))
		},
	})

	stdout, _, err := runContractCommand(t, url, []string{"related-artifacts"}, nil, false,
		map[string]string{"kind": "workflow", "id": relatedWorkflowID})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"KIND", "Anchor", "(this artifact)", "Triage agent", "(no access)"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q: %q", want, stdout)
		}
	}
}
