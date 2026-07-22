package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/major-technology/cli/projects"
	"github.com/spf13/cobra"
)

// compileResultRequest is the body of POST /internal/projects/compile-result.
type compileResultRequest struct {
	ProjectID       string          `json:"projectId"`
	OrganizationID  string          `json:"organizationId"`
	CommitHash      string          `json:"commitHash"`
	CompilerVersion string          `json:"compilerVersion"`
	Status          string          `json:"status"` // "compiled" | "failed"
	CompiledConfig  json.RawMessage `json:"compiledConfig,omitempty"`
	CompileError    string          `json:"compileError,omitempty"`
}

// buildCompileResult compiles dir and shapes the outcome (success or
// validation failure) into a report body. Validation failures are successful
// reports: the platform stores them as failed versions.
func buildCompileResult(dir, projectID, orgID, commitHash, compilerVersion string) compileResultRequest {
	body := compileResultRequest{
		ProjectID:       projectID,
		OrganizationID:  orgID,
		CommitHash:      commitHash,
		CompilerVersion: compilerVersion,
	}

	result, issues := projects.Compile(dir)

	if len(issues) > 0 {
		var lines []string
		for _, issue := range issues {
			location := issue.File
			if issue.Path != "" {
				location += " " + issue.Path
			}
			lines = append(lines, location+": "+issue.Message)
		}

		body.Status = "failed"
		body.CompileError = strings.Join(lines, "\n")
		return body
	}

	body.Status = "compiled"
	body.CompiledConfig = result.ConfigJSON
	return body
}

// postCompileResult delivers the report with the cross-server JWT. The
// server's cross-server auth middleware reads the JWT off a custom header
// (x-major-cross-server-jwt), not Authorization - this matches every other
// internal caller in the monorepo (go-api, go-auth, go-temporal-worker, ...).
func postCompileResult(apiURL, jwt string, body compileResultRequest) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	req, err := http.NewRequest("POST", strings.TrimRight(apiURL, "/")+"/internal/projects/compile-result", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build report request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-major-cross-server-jwt", jwt)

	client := &http.Client{Timeout: 60 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver report: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("report rejected with status %d", resp.StatusCode)
	}

	return nil
}

// cloneAtCommit fetches exactly one commit of the repo into dir using the
// installation token over HTTPS.
func cloneAtCommit(repoURL, token, commitHash, dir string) error {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("bad REPO_URL: %w", err)
	}
	parsed.User = url.UserPassword("x-access-token", token)

	steps := [][]string{
		{"git", "init", "--quiet", dir},
		{"git", "-C", dir, "fetch", "--quiet", "--depth", "1", parsed.String(), commitHash},
		{"git", "-C", dir, "checkout", "--quiet", "FETCH_HEAD"},
	}

	for _, step := range steps {
		cmd := exec.Command(step[0], step[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			// Never echo the tokenized URL.
			label := sanitize(strings.Join(step, " "), token)
			return fmt.Errorf("%s failed: %s", label, sanitize(string(out), token))
		}
	}

	return nil
}

func sanitize(s, secret string) string {
	if secret == "" {
		return s
	}
	return strings.ReplaceAll(s, secret, "***")
}

func newCompileAndReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "compile-and-report",
		Short:  "Clone, compile, and report a project version (internal)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			required := []string{"REPO_URL", "GITHUB_TOKEN", "COMMIT_HASH", "PROJECT_ID", "ORGANIZATION_ID", "API_URL", "CROSS_SERVER_JWT"}
			env := map[string]string{}

			for _, key := range required {
				value := os.Getenv(key)
				if value == "" {
					return fmt.Errorf("missing required env var %s", key)
				}
				env[key] = value
			}

			workDir, err := os.MkdirTemp("", "project-compile-")
			if err != nil {
				return fmt.Errorf("create workdir: %w", err)
			}
			defer os.RemoveAll(workDir)

			if err := cloneAtCommit(env["REPO_URL"], env["GITHUB_TOKEN"], env["COMMIT_HASH"], workDir); err != nil {
				return err
			}

			body := buildCompileResult(workDir, env["PROJECT_ID"], env["ORGANIZATION_ID"], env["COMMIT_HASH"], CLIVersion)

			if err := postCompileResult(env["API_URL"], env["CROSS_SERVER_JWT"], body); err != nil {
				return err
			}

			cmd.Printf("reported compile result: %s (commit %s)\n", body.Status, env["COMMIT_HASH"])
			return nil
		},
	}

	return cmd
}
