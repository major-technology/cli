package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	clierrors "github.com/major-technology/cli/errors"
)

func init() {
	testTokenOverride = "test-token"
}

// newTestServer spins an httptest server and points a Client at it. Requests
// authenticate with the keyring token; tests bypass auth via testTokenOverride
// so doRequest's auth path still runs, but against the stub token.
func newTestServer(t *testing.T, wantMethod, wantPath string, status int, respBody any) (*httptest.Server, *Client) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod {
			t.Errorf("method = %s, want %s", r.Method, wantMethod)
		}
		if r.URL.RequestURI() != wantPath {
			t.Errorf("path = %s, want %s", r.URL.RequestURI(), wantPath)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	t.Cleanup(srv.Close)

	return srv, NewClient(srv.URL)
}

func TestCreateProjectRequestShape(t *testing.T) {
	_, client := newTestServer(t, "POST", "/projects", 200, CreateProjectResponse{
		ProjectID:      "p-1",
		RepositoryName: "staging-org-proj",
		CloneURLSSH:    "git@github.com:major-technology/staging-org-proj.git",
		CloneURLHTTPS:  "https://github.com/major-technology/staging-org-proj.git",
	})

	resp, err := client.CreateProject("proj", "desc", "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProjectID != "p-1" || resp.RepositoryName == "" {
		t.Fatalf("bad response mapping: %+v", resp)
	}
}

func TestGetProjectByRepoRequestShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.RequestURI() != "/projects/from-repo" {
			t.Errorf("path = %s, want /projects/from-repo", r.URL.RequestURI())
		}
		var body GetProjectByRepoRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("bad body: %v", err)
		}
		if body.Owner != "major-technology" || body.Repo != "staging-org-proj" {
			t.Errorf("body = %+v, want owner/repo", body)
		}
		_ = json.NewEncoder(w).Encode(GetProjectByRepoResponse{
			ProjectID:      "p-1",
			OrganizationID: "org-1",
			Name:           "proj",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	resp, err := client.GetProjectByRepo("major-technology", "staging-org-proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProjectID != "p-1" || resp.OrganizationID != "org-1" || resp.Name != "proj" {
		t.Fatalf("bad response mapping: %+v", resp)
	}
}

func TestGetProjectPath(t *testing.T) {
	_, client := newTestServer(t, "GET", "/projects/p-1", 200, GetProjectResponse{
		ProjectID:      "p-1",
		Name:           "proj",
		RepositoryName: "staging-org-proj",
		LatestVersion: &ProjectVersionItem{
			ID:            "v-9",
			CommitHash:    "abc123",
			CompileStatus: "success",
			CreatedAt:     "2026-07-21T00:00:00Z",
		},
	})

	resp, err := client.GetProject("p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProjectID != "p-1" || resp.LatestVersion == nil || resp.LatestVersion.ID != "v-9" {
		t.Fatalf("bad response mapping: %+v", resp)
	}
}

func TestListProjectVersionsPath(t *testing.T) {
	_, client := newTestServer(t, "GET", "/projects/p-1/versions", 200, ListProjectVersionsResponse{
		Versions: []ProjectVersionItem{
			{ID: "v-1", CommitHash: "aaa", CompileStatus: "success", CreatedAt: "2026-07-20T00:00:00Z"},
			{ID: "v-2", CommitHash: "bbb", CompileStatus: "failed", CompileError: "bad schema", CreatedAt: "2026-07-21T00:00:00Z"},
		},
	})

	resp, err := client.ListProjectVersions("p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Versions) != 2 {
		t.Fatalf("bad versions mapping: %+v", resp)
	}
}

func TestGetProjectDeployPlanPath(t *testing.T) {
	_, client := newTestServer(t, "GET", "/projects/p-1/deploy-plan?versionId=v-9", 200, GetProjectDeployPlanResponse{
		Creates: []string{"triage"},
		Deletes: []string{"old"},
	})

	resp, err := client.GetProjectDeployPlan("p-1", "v-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Creates) != 1 || len(resp.Deletes) != 1 {
		t.Fatalf("bad plan mapping: %+v", resp)
	}
}

func TestCreateProjectDeployBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.RequestURI() != "/projects/p-1/deploys" {
			t.Errorf("path = %s, want /projects/p-1/deploys", r.URL.RequestURI())
		}
		var body CreateProjectDeployRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("bad body: %v", err)
		}
		if body.ProjectVersionID != "v-9" {
			t.Errorf("projectVersionId = %q, want v-9", body.ProjectVersionID)
		}
		_ = json.NewEncoder(w).Encode(CreateProjectDeployResponse{DeployID: "d-1", Status: "deployed"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	resp, err := client.CreateProjectDeploy("p-1", "v-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "deployed" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestAddProjectGithubCollaboratorsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.RequestURI() != "/projects/p-1/add-gh-collaborators" {
			t.Errorf("path = %s, want /projects/p-1/add-gh-collaborators", r.URL.RequestURI())
		}
		var body AddProjectGithubCollaboratorsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("bad body: %v", err)
		}
		if body.GithubUsername != "octocat" {
			t.Errorf("githubUsername = %q, want octocat", body.GithubUsername)
		}
		_ = json.NewEncoder(w).Encode(AddGithubCollaboratorsResponse{Success: true, Message: "added"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	resp, err := client.AddProjectGithubCollaborators("p-1", "octocat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("bad response mapping: %+v", resp)
	}
}

func TestProjectEndpointsErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
		call   func(client *Client) error
	}{
		{
			name:   "GetProject not found",
			status: http.StatusNotFound,
			call: func(client *Client) error {
				_, err := client.GetProject("missing")
				return err
			},
		},
		{
			name:   "CreateProject unauthorized",
			status: http.StatusUnauthorized,
			call: func(client *Client) error {
				_, err := client.CreateProject("proj", "desc", "org-1")
				return err
			},
		},
		{
			name:   "GetProjectByRepo not found",
			status: http.StatusNotFound,
			call: func(client *Client) error {
				_, err := client.GetProjectByRepo("owner", "repo")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(ErrorResponse{
					Error: &AppErrorDetail{
						InternalCode: 9999,
						ErrorString:  "boom",
						StatusCode:   tt.status,
					},
				})
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			err := tt.call(client)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			cliErr, ok := err.(*clierrors.CLIError)
			if !ok {
				t.Fatalf("error type = %T, want *clierrors.CLIError", err)
			}
			// Unmapped internal codes fall through to the generic CLIError,
			// which must retain the originating HTTP status so callers (e.g.
			// cmd/project's isProjectNotFoundError) can branch on a 404
			// without the fallback Title ("API Error (Code: 9999)") hiding it.
			if cliErr.StatusCode != tt.status {
				t.Fatalf("StatusCode = %d, want %d", cliErr.StatusCode, tt.status)
			}
		})
	}
}
