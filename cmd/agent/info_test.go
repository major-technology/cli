package agent

import (
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
)

func TestLocalAgentVersionReadsTheAgentFolder(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "agent", AgentID: testAgentID, Version: 2}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	version, ok := localAgentVersion(testAgentID)
	if !ok || version != 2 {
		t.Fatalf("version=%d ok=%v", version, ok)
	}
}

func TestLocalAgentVersionIgnoresAnotherAgent(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "agent", AgentID: testAgentID, Version: 2}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if _, ok := localAgentVersion("22222222-2222-4222-8222-222222222222"); ok {
		t.Fatal("expected no local version for a different agent")
	}
}

func TestLocalAgentVersionIgnoresAFolderWithNoRecordedVersion(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Write(dir, workspace.Config{OrganizationID: "org-1", Target: workspace.Target{Kind: "agent", AgentID: testAgentID}}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if _, ok := localAgentVersion(testAgentID); ok {
		t.Fatal("expected no local version for a folder written before versions were recorded")
	}
}

func TestLocalAgentVersionIgnoresANonAgentFolder(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, ok := localAgentVersion(testAgentID); ok {
		t.Fatal("expected no local version outside an agent folder")
	}
}

func TestInfoResultStringWithoutLocalVersion(t *testing.T) {
	text := infoResult{AgentInfoResponse: &api.AgentInfoResponse{ID: testAgentID, Name: "Helper"}}.String()
	if strings.Contains(text, "Local version") {
		t.Fatalf("unexpected local version line: %q", text)
	}
}

func TestInfoResultStringUpToDate(t *testing.T) {
	result := infoResult{
		AgentInfoResponse: &api.AgentInfoResponse{ID: testAgentID, Name: "Helper", LatestVersion: &api.AgentLatestVersion{Version: 3}},
		hasLocal:          true,
		localVersion:      3,
	}
	if text := result.String(); !strings.Contains(text, "Local version: 3 (up to date)") {
		t.Fatalf("text: %q", text)
	}
}

func TestInfoResultStringBehindLatest(t *testing.T) {
	result := infoResult{
		AgentInfoResponse: &api.AgentInfoResponse{ID: testAgentID, Name: "Helper", LatestVersion: &api.AgentLatestVersion{Version: 3}},
		hasLocal:          true,
		localVersion:      2,
	}
	if text := result.String(); !strings.Contains(text, "Local version: 2 (behind latest 3; run major pull)") {
		t.Fatalf("text: %q", text)
	}
}
