package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/major-technology/cli/clients/git"
)

// GenerateMcpConfig generates a .mcp.json file for Claude Code in the specified directory.
// If targetDir is empty, it uses the current git repository root.
// It uses the env vars from the application env endpoint to construct the MCP server config
// pointing to the Go API's resource MCP endpoint.
func GenerateMcpConfig(targetDir string, envVars map[string]string) (string, error) {
	apiBaseURL := envVars["MAJOR_API_BASE_URL"]
	jwtToken := envVars["MAJOR_JWT_TOKEN"]
	applicationID := envVars["APPLICATION_ID"]

	if apiBaseURL == "" || jwtToken == "" || applicationID == "" {
		return "", fmt.Errorf("missing required env vars for MCP config (MAJOR_API_BASE_URL, MAJOR_JWT_TOKEN, APPLICATION_ID)")
	}

	if targetDir == "" {
		var err error
		targetDir, err = git.GetRepoRoot()
		if err != nil {
			return "", fmt.Errorf("failed to get git repository root: %w", err)
		}
	}

	mcpURL := fmt.Sprintf("%s/internal/apps/v1/%s/mcp", apiBaseURL, applicationID)

	config := map[string]any{
		"mcpServers": map[string]any{
			"major": map[string]any{
				"type": "http",
				"url":  mcpURL,
				"headers": map[string]string{
					"x-major-jwt": jwtToken,
				},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	jsonBytes = append(jsonBytes, '\n')

	mcpPath := filepath.Join(targetDir, ".mcp.json")
	if err := os.WriteFile(mcpPath, jsonBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write .mcp.json file: %w", err)
	}

	// Ensure .mcp.json is in .gitignore
	ensureGitignoreEntry(targetDir, ".mcp.json")

	return mcpPath, nil
}

// ensureGitignoreEntry appends an entry to .gitignore if it's not already present.
func ensureGitignoreEntry(dir, entry string) {
	gitignorePath := filepath.Join(dir, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		// No .gitignore file, create one
		_ = os.WriteFile(gitignorePath, []byte(entry+"\n"), 0644)
		return
	}

	// Check if entry already exists
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return
		}
	}

	// Append the entry with a preceding newline if file doesn't end with one
	suffix := entry + "\n"
	if len(content) > 0 && content[len(content)-1] != '\n' {
		suffix = "\n" + suffix
	}
	_ = os.WriteFile(gitignorePath, append(content, []byte(suffix)...), 0644)
}
