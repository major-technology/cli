package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	configDirName  = ".major"
	configFileName = "config.json"
	excludeRule    = "/.major/config.json"
)

// uuidPattern matches the CLI repo's existing UUID contract (bindings.schema.json):
// RFC 4122 versions 1-8 with RFC 4122 variant, without adding a UUID dependency.
var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

var kindIDField = map[string]string{
	"app":      "applicationId",
	"skill":    "skillId",
	"agent":    "agentId",
	"workflow": "workflowId",
}

// ErrNotFound is returned when no .major/config.json exists before a Git or
// filesystem boundary.
var ErrNotFound = errors.New("workspace config not found")

type Target struct {
	Kind          string `json:"kind"`
	ApplicationID string `json:"applicationId,omitempty"`
	SkillID       string `json:"skillId,omitempty"`
	AgentID       string `json:"agentId,omitempty"`
	WorkflowID    string `json:"workflowId,omitempty"`
}

type Config struct {
	OrganizationID string `json:"organizationId"`
	Target         Target `json:"target"`
}

func (t Target) Validate() error {
	ids := map[string]string{
		"app":      t.ApplicationID,
		"skill":    t.SkillID,
		"agent":    t.AgentID,
		"workflow": t.WorkflowID,
	}

	var present []string
	for kind, id := range ids {
		if id != "" {
			present = append(present, kind)
		}
	}
	if len(present) != 1 {
		return fmt.Errorf("target must have exactly one of applicationId, skillId, agentId, or workflowId")
	}
	if _, ok := kindIDField[t.Kind]; !ok {
		return fmt.Errorf("unknown target kind %q", t.Kind)
	}
	if present[0] != t.Kind {
		return fmt.Errorf("target kind %q must use %s, not %s", t.Kind, kindIDField[t.Kind], kindIDField[present[0]])
	}
	if !uuidPattern.MatchString(ids[t.Kind]) {
		return fmt.Errorf("invalid %s UUID", kindIDField[t.Kind])
	}
	return nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.OrganizationID) == "" {
		return fmt.Errorf("organizationId is required")
	}
	return c.Target.Validate()
}

func Load(startDir string) (*Config, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	for {
		path := filepath.Join(dir, configDirName, configFileName)
		data, err := os.ReadFile(path)
		if err == nil {
			cfg, parseErr := parseConfig(path, data)
			if parseErr != nil {
				return nil, parseErr
			}
			return cfg, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}

		if _, statErr := os.Lstat(filepath.Join(dir, ".git")); statErr == nil {
			return nil, ErrNotFound
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotFound
		}
		dir = parent
	}
}

func Write(projectDir string, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}

	dir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	configDir := filepath.Join(dir, configDirName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(configDir, configFileName)
	var existing []byte
	if data, err := os.ReadFile(path); err == nil {
		existing = data
	} else if !os.IsNotExist(err) {
		return err
	}

	body, err := encodeConfig(existing, config)
	if err != nil {
		return invalidConfigError(path, err)
	}

	tmp, err := os.CreateTemp(configDir, "config.json.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func IgnoreLocalConfig(projectDir string) error {
	dir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "rev-parse", "--git-path", "info/exclude")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if isNotGitRepository(out, err) {
			return nil
		}
		return fmt.Errorf("git rev-parse --git-path info/exclude: %w", err)
	}

	exclude := strings.TrimSpace(string(out))
	if exclude == "" {
		return nil
	}
	if !filepath.IsAbs(exclude) {
		exclude = filepath.Join(dir, exclude)
	}

	if err := os.MkdirAll(filepath.Dir(exclude), 0755); err != nil {
		return err
	}

	var content []byte
	if data, err := os.ReadFile(exclude); err == nil {
		content = data
	} else if !os.IsNotExist(err) {
		return err
	}

	if hasExcludeRule(content) {
		return nil
	}

	updated := appendExcludeRule(content)
	return os.WriteFile(exclude, updated, 0644)
}

func parseConfig(path string, data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, invalidConfigError(path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, invalidConfigError(path, err)
	}
	return &cfg, nil
}

func encodeConfig(existing []byte, cfg Config) ([]byte, error) {
	overlayBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var overlay map[string]json.RawMessage
	if err := json.Unmarshal(overlayBytes, &overlay); err != nil {
		return nil, err
	}

	merged := map[string]json.RawMessage{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &merged); err != nil {
			return nil, err
		}
		if merged == nil {
			return nil, fmt.Errorf("config root must be a JSON object")
		}
	}
	for key, value := range overlay {
		merged[key] = value
	}

	body, err := json.MarshalIndent(merged, "", "\t")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func invalidConfigError(path string, err error) error {
	return fmt.Errorf("invalid workspace config %s: %w; repair .major/config.json with organizationId and exactly one app, skill, agent, or workflow target", path, err)
}

func isNotGitRepository(out []byte, err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	msg := strings.ToLower(string(out) + err.Error())
	return strings.Contains(msg, "not a git repository")
}

func hasExcludeRule(content []byte) bool {
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == excludeRule {
			return true
		}
	}
	return false
}

func appendExcludeRule(content []byte) []byte {
	text := string(content)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return []byte(text + excludeRule + "\n")
}
