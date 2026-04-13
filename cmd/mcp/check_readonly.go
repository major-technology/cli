package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var checkReadonlyCmd = &cobra.Command{
	Use:    "check-readonly [tool-name]",
	Short:  "Check if an MCP tool is read-only",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckReadonly(args[0])
	},
}

var checkReadonlyHookCmd = &cobra.Command{
	Use:    "check-readonly-hook",
	Short:  "Hook entry point: reads PreToolUse JSON from stdin",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, err := io.ReadAll(os.Stdin)
		if err != nil || len(input) == 0 {
			return nil
		}

		var payload struct {
			ToolName string `json:"tool_name"`
		}
		if err := json.Unmarshal(input, &payload); err != nil || payload.ToolName == "" {
			return nil
		}

		// Strip MCP server prefix
		actualTool := payload.ToolName
		const prefix = "mcp__plugin_major_major-resources__"
		if strings.HasPrefix(actualTool, prefix) {
			actualTool = actualTool[len(prefix):]
		}

		return runCheckReadonly(actualTool)
	},
}

type toolMetadataItem struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ReadOnly     bool   `json:"readOnly"`
	ResourceType string `json:"resourceType"`
}

type toolCache struct {
	Tools     []toolMetadataItem `json:"tools"`
	FetchedAt time.Time          `json:"fetchedAt"`
}

func runCheckReadonly(toolName string) error {
	tools, err := getCachedToolMetadata()
	if err != nil {
		// If we can't get metadata, don't block — let Claude Code prompt the user
		return nil
	}

	for _, t := range tools {
		if t.Name == toolName && t.ReadOnly {
			result := map[string]any{
				"hookSpecificOutput": map[string]any{
					"hookEventName":      "PreToolUse",
					"permissionDecision": "allow",
				},
			}

			return json.NewEncoder(os.Stdout).Encode(result)
		}
	}

	// Not read-only or not found — exit silently, Claude Code will prompt the user
	return nil
}

func getCachedToolMetadata() ([]toolMetadataItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(homeDir, ".major", "cache", "tool-metadata.json")

	// Try to read cache
	data, err := os.ReadFile(cachePath)
	if err == nil {
		var cache toolCache

		if json.Unmarshal(data, &cache) == nil {
			if time.Since(cache.FetchedAt) < 24*time.Hour {
				return cache.Tools, nil
			}
		}
	}

	// Fetch fresh metadata from resource API
	cfg := singletons.GetConfig()
	if cfg == nil || cfg.ResourceAPIURL == "" {
		return nil, fmt.Errorf("config not loaded")
	}

	url := cfg.ResourceAPIURL + "/cli/v1/mcp/tools"

	// Get auth headers
	token, err := mjrToken.GetToken()
	if err != nil {
		return nil, err
	}

	orgID, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-major-org-id", orgID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tools endpoint returned %d", resp.StatusCode)
	}

	var toolsResp struct {
		Tools []toolMetadataItem `json:"tools"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&toolsResp); err != nil {
		return nil, err
	}

	// Cache it
	cache := toolCache{
		Tools:     toolsResp.Tools,
		FetchedAt: time.Now(),
	}

	cacheData, _ := json.Marshal(cache)
	os.MkdirAll(filepath.Dir(cachePath), 0755)
	os.WriteFile(cachePath, cacheData, 0644)

	return toolsResp.Tools, nil
}
