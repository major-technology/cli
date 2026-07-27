package projects

// Skill discovery and bundle validation/publishing. Discovery here mirrors
// exactly how Load discovers agents under <srcDir>/agents (load.go): a
// directory is a skill only if it contains a marker file (SKILL.md instead of
// agent.json); directories without one are silently ignored. Bundle
// validation (file walk, size caps, frontmatter) is a Go port of the server's
// validator: packages/shared/src/utils/skill-validator.ts. Compiling never
// inlines file contents: a skill compiles down to the git tree hash of its
// directory at HEAD, plus an optional zip written to --skill-bundles-dir for
// the mono-owned compile job to upload to S3 and inject as the "bundle"
// field before submitting - the CLI itself never touches S3.

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// SkillSlugPattern matches the server's skill slug contract: lowercase
// kebab-case, 1-63 characters (packages/api/src/schemas/projects.ts).
var SkillSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Skill bundle caps, ported from SKILL_VALIDATION_LIMITS
// (packages/shared/src/constants/skill-validation.ts). The file extension
// allow-list the server used to enforce is gone: bundles may now contain
// arbitrary binary files (fonts, images, ...), so any extension is accepted.
const (
	MaxSkillSlugLen        = 63
	MaxSkillFiles          = 200
	MaxSkillFileBytes      = 5 * 1024 * 1024
	MaxSkillBundleBytes    = 25 * 1024 * 1024
	MaxSkillPathLen        = 512
	MaxSkillNameLen        = 120
	MaxSkillDescriptionLen = 2000
)

const skillGitkeep = ".gitkeep"

// discoverSkills finds skill directories under <srcDir>/skills, mirroring
// how Load discovers agents under <srcDir>/agents: a directory is a skill
// only if it contains a SKILL.md; directories without one are ignored (same
// convention as agent directories missing agent.json).
func discoverSkills(dir, srcDir string) ([]SkillDefinition, []Issue) {
	skillsDir := filepath.Join(dir, srcDir, "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, nil // no skills directory means no skills, not an error
	}

	var skills []SkillDefinition
	var issues []Issue
	seenSlugs := map[string]string{} // slug -> skill dir that declared it

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(srcDir, "skills", entry.Name())
		skillMDFile := filepath.Join(skillDir, "SKILL.md")

		if _, err := os.Stat(filepath.Join(dir, skillMDFile)); err != nil {
			continue // directories without SKILL.md are ignored
		}

		slug := entry.Name()
		if !SkillSlugPattern.MatchString(slug) || len(slug) > MaxSkillSlugLen {
			issues = append(issues, Issue{
				File:    skillMDFile,
				Message: fmt.Sprintf("invalid skill slug %q: must match ^[a-z0-9]+(-[a-z0-9]+)*$ and be 1-%d characters", slug, MaxSkillSlugLen),
			})
			continue
		}

		if prevDir, dup := seenSlugs[slug]; dup {
			issues = append(issues, Issue{
				File:    skillMDFile,
				Message: fmt.Sprintf("duplicate skill slug %q (already declared in %s)", slug, prevDir),
			})
			continue
		}
		seenSlugs[slug] = skillDir

		skills = append(skills, SkillDefinition{Slug: slug, Dir: skillDir})
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Slug < skills[j].Slug })

	return skills, issues
}

// skillFile is one file collected while walking a skill directory. content is
// only populated for SKILL.md (frontmatter needs it as text) and, when a
// bundle zip is being written, for every file - raw bytes, never decoded.
type skillFile struct {
	relPath string
	size    int64
	content []byte
}

// readSkillBundle walks a skill directory, validates every file against the
// server's skill-validator rules, and returns the compiled skill: its slug
// and the git tree hash of its directory at HEAD. When skillBundlesDir is
// non-empty, it also writes <skillBundlesDir>/<treeHash>.zip with every file's
// raw bytes. Symlinks (files or dirs) are rejected, mirroring readPromptFile's
// symlink handling in prompt.go.
func readSkillBundle(projectDir string, skill SkillDefinition, skillBundlesDir string) (*CompiledSkill, []Issue) {
	root := filepath.Join(projectDir, skill.Dir)
	needAllContent := skillBundlesDir != ""

	var files []skillFile
	var issues []Issue

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		relPath := filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
		fileID := filepath.Join(skill.Dir, filepath.FromSlash(relPath))

		if d.Type()&fs.ModeSymlink != 0 {
			issues = append(issues, Issue{
				File:    fileID,
				Message: fmt.Sprintf("skill %q: %q is a symlink; symlinks are not allowed", skill.Slug, relPath),
			})
			return nil
		}

		if d.IsDir() {
			return nil
		}

		issues = append(issues, validateSkillFilePath(skill.Slug, fileID, relPath)...)

		info, err := d.Info()
		if err != nil {
			return err
		}

		if info.Size() > MaxSkillFileBytes {
			issues = append(issues, Issue{
				File:    fileID,
				Message: fmt.Sprintf("skill %q: %q exceeds the per-file size cap of %d bytes", skill.Slug, relPath, MaxSkillFileBytes),
			})
		}

		if utf8.RuneCountInString(relPath) > MaxSkillPathLen {
			issues = append(issues, Issue{
				File:    fileID,
				Message: fmt.Sprintf("skill %q: path %q exceeds the max length of %d characters", skill.Slug, relPath, MaxSkillPathLen),
			})
		}

		sf := skillFile{relPath: relPath, size: info.Size()}

		// Only SKILL.md is ever read as text (frontmatter parsing below); every
		// other file's bytes are read only if a bundle zip needs them, and are
		// never treated as UTF-8.
		if relPath == "SKILL.md" || needAllContent {
			content, err := os.ReadFile(path)
			if err != nil {
				issues = append(issues, Issue{File: fileID, Message: "cannot read file: " + err.Error()})
				return nil
			}
			sf.content = content
		}

		files = append(files, sf)

		return nil
	})
	if walkErr != nil {
		return nil, []Issue{{File: skill.Dir, Message: fmt.Sprintf("cannot read skill %q: %s", skill.Slug, walkErr.Error())}}
	}

	if len(files) > MaxSkillFiles {
		issues = append(issues, Issue{
			File:    skill.Dir,
			Message: fmt.Sprintf("skill %q exceeds the max file count of %d", skill.Slug, MaxSkillFiles),
		})
	}

	var totalSize int64
	var skillMD *skillFile
	for i := range files {
		totalSize += files[i].size
		if files[i].relPath == "SKILL.md" {
			skillMD = &files[i]
		}
	}

	if totalSize > MaxSkillBundleBytes {
		issues = append(issues, Issue{
			File:    skill.Dir,
			Message: fmt.Sprintf("skill %q exceeds the max bundle size of %d bytes", skill.Slug, MaxSkillBundleBytes),
		})
	}

	if skillMD == nil {
		issues = append(issues, Issue{
			File:    skill.Dir,
			Message: fmt.Sprintf("skill %q must contain SKILL.md at its root", skill.Slug),
		})
	} else {
		issues = append(issues, validateSkillFrontmatter(skill.Slug, filepath.Join(skill.Dir, "SKILL.md"), skillMD.content)...)
	}

	treeHash, hashIssues := resolveSkillTreeHash(projectDir, skill)
	issues = append(issues, hashIssues...)

	if len(issues) > 0 {
		return nil, issues
	}

	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })

	if needAllContent {
		if err := writeSkillBundleZip(skillBundlesDir, treeHash, files); err != nil {
			return nil, []Issue{{File: skill.Dir, Message: fmt.Sprintf("cannot write bundle zip for skill %q: %s", skill.Slug, err.Error())}}
		}
	}

	return &CompiledSkill{Slug: skill.Slug, TreeHash: treeHash}, nil
}

// writeSkillBundleZip writes every file in files to
// <bundlesDir>/<treeHash>.zip: raw bytes, forward-slash paths relative to the
// skill directory, no transformation. Only called locally (--skill-bundles-dir);
// the authoritative compile job builds and uploads its own bundle from a
// clean clone.
func writeSkillBundleZip(bundlesDir, treeHash string, files []skillFile) error {
	if err := os.MkdirAll(bundlesDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(bundlesDir, treeHash+".zip"))
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, sf := range files {
		w, err := zw.Create(sf.relPath)
		if err != nil {
			return err
		}
		if _, err := w.Write(sf.content); err != nil {
			return err
		}
	}

	return zw.Close()
}

// skillTreeHashPattern matches a git tree object hash: 40 hex characters for
// SHA-1 repositories, 64 for SHA-256 ones.
var skillTreeHashPattern = regexp.MustCompile(`^[0-9a-f]{40}$|^[0-9a-f]{64}$`)

// runGit runs git with args in dir and returns trimmed stdout, or an error on
// failure (matching clients/git's style of shelling out per-call rather than
// keeping a long-lived process).
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// resolveSkillTreeHash computes the compiled tree hash for a skill directory:
// the git tree object hash of the directory at HEAD (git rev-parse
// HEAD:<repo-relative-path>). If the project isn't inside a git repository,
// or the skill directory has never been committed, that's a validation issue
// telling the user to commit it first. If the skill directory has uncommitted
// changes, the hash still resolves (it reflects HEAD), but that's reported as
// an issue too - Issue has no severity levels, so this is a hard issue, not
// merely a warning; the authoritative compile always runs on a clean clone at
// a fixed commit, so in practice this only affects local runs.
func resolveSkillTreeHash(projectDir string, skill SkillDefinition) (string, []Issue) {
	absSkillDir := filepath.Join(projectDir, skill.Dir)

	repoRoot, err := runGit(absSkillDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", []Issue{{
			File:    skill.Dir,
			Message: fmt.Sprintf("skill %q is not inside a git repository; commit the project, including the skill directory, before compiling", skill.Slug),
		}}
	}

	// git resolves symlinks in --show-toplevel; resolve absSkillDir the same
	// way (macOS puts TMPDIR behind a /var -> /private/var symlink) so the two
	// paths are comparable instead of producing a bogus "../../.." relative
	// path that still resolves but never matches HEAD.
	realSkillDir, err := filepath.EvalSymlinks(absSkillDir)
	if err != nil {
		return "", []Issue{{File: skill.Dir, Message: "cannot resolve skill directory path: " + err.Error()}}
	}

	relToRepo, err := filepath.Rel(repoRoot, realSkillDir)
	if err != nil {
		return "", []Issue{{File: skill.Dir, Message: "cannot resolve skill directory relative to the git repository root: " + err.Error()}}
	}
	relToRepo = filepath.ToSlash(relToRepo)

	hash, err := runGit(repoRoot, "rev-parse", "HEAD:"+relToRepo)
	if err != nil || !skillTreeHashPattern.MatchString(hash) {
		return "", []Issue{{
			File:    skill.Dir,
			Message: fmt.Sprintf("skill %q has never been committed to git; commit the skill directory (%s) before compiling", skill.Slug, relToRepo),
		}}
	}

	var issues []Issue
	if status, statusErr := runGit(repoRoot, "status", "--porcelain", "--", relToRepo); statusErr == nil && status != "" {
		issues = append(issues, Issue{
			File:    skill.Dir,
			Message: fmt.Sprintf("skill %q has uncommitted changes; the compiled tree hash reflects the last commit (HEAD), not the working tree", skill.Slug),
		})
	}

	return hash, issues
}

// validateSkillFilePath ports the path-level checks from the server's
// validateSkillFilePath (packages/shared/src/utils/skill-validator.ts):
// backslashes and null bytes are defensive rejections, and dotfiles are
// rejected except .gitkeep. The extension allow-list this used to enforce is
// gone - any extension is accepted, since bundles may contain binary assets.
func validateSkillFilePath(slug, fileID, relPath string) []Issue {
	var issues []Issue

	if strings.Contains(relPath, "\x00") {
		issues = append(issues, Issue{File: fileID, Message: fmt.Sprintf("skill %q: path %q contains a null byte", slug, relPath)})
	}

	if strings.Contains(relPath, "\\") {
		issues = append(issues, Issue{File: fileID, Message: fmt.Sprintf("skill %q: path %q contains a backslash; use forward slashes", slug, relPath)})
	}

	basename := relPath
	if idx := strings.LastIndex(relPath, "/"); idx >= 0 {
		basename = relPath[idx+1:]
	}

	if strings.HasPrefix(basename, ".") && basename != skillGitkeep {
		issues = append(issues, Issue{File: fileID, Message: fmt.Sprintf("skill %q: dotfile %q is not allowed (only %s is)", slug, basename, skillGitkeep)})
	}

	return issues
}

// skillFrontmatterRe matches a leading --- delimited YAML block, mirroring
// the server's /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/.
var skillFrontmatterRe = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---\r?\n?`)

// skillFrontmatterFieldRe matches a "key: value" line.
var skillFrontmatterFieldRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)\s*:\s*(.*)$`)

var skillFrontmatterLineSplit = regexp.MustCompile(`\r?\n`)

// validateSkillFrontmatter ports parseSkillFrontmatter
// (packages/shared/src/utils/skill-validator.ts): a lightweight line-based
// reader for the two scalar fields SKILL.md must declare, no YAML library.
// content is SKILL.md's bytes - the one file in a skill bundle ever decoded
// as text.
func validateSkillFrontmatter(slug, skillMDFile string, content []byte) []Issue {
	match := skillFrontmatterRe.FindSubmatch(content)
	if match == nil {
		return []Issue{{
			File:    skillMDFile,
			Message: fmt.Sprintf("skill %q: SKILL.md must begin with a YAML frontmatter block delimited by --- lines", slug),
		}}
	}

	body := string(match[1])
	fields := map[string]string{}

	for _, line := range skillFrontmatterLineSplit.Split(body, -1) {
		m := skillFrontmatterFieldRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		fields[m[1]] = unquoteSkillFrontmatterValue(strings.TrimSpace(m[2]))
	}

	var issues []Issue

	name := strings.TrimSpace(fields["name"])
	if name == "" || utf8.RuneCountInString(name) > MaxSkillNameLen {
		issues = append(issues, Issue{
			File:    skillMDFile,
			Message: fmt.Sprintf("skill %q: frontmatter \"name\" must be a non-empty string up to %d chars", slug, MaxSkillNameLen),
		})
	}

	description := strings.TrimSpace(fields["description"])
	if description == "" || utf8.RuneCountInString(description) > MaxSkillDescriptionLen {
		issues = append(issues, Issue{
			File:    skillMDFile,
			Message: fmt.Sprintf("skill %q: frontmatter \"description\" must be a non-empty string up to %d chars", slug, MaxSkillDescriptionLen),
		})
	}

	return issues
}

// unquoteSkillFrontmatterValue strips matching outer quotes and unescapes
// per-quote-style escapes, mirroring the server's parseSkillFrontmatter:
// double quotes unescape \" and \\, single quotes unescape ”.
func unquoteSkillFrontmatterValue(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		inner := value[1 : len(value)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner
	}

	if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		inner := value[1 : len(value)-1]
		return strings.ReplaceAll(inner, "''", "'")
	}

	return value
}
