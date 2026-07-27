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

		skill := SkillDefinition{Slug: slug, Dir: skillDir}

		// Content-only validation runs as part of discovery (Load), so
		// `validate` catches dotfiles, size caps, bad frontmatter, etc. It
		// never touches git *state* - only listSkillFiles's file-listing
		// path uses git, to exclude gitignored files - so an uncommitted or
		// never-committed skill directory validates cleanly. Git state
		// (tree-hash resolution) is a compile-only concern (readSkillBundle).
		_, contentIssues := validateSkillContent(dir, skill, false)
		issues = append(issues, contentIssues...)

		skills = append(skills, skill)
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

// listSkillFiles resolves the relative file paths under a skill directory.
// Inside a git repository, the list comes from `git ls-files -co
// --exclude-standard`: tracked files plus untracked-but-not-ignored ones, so
// a gitignored file (e.g. .DS_Store) never reaches validation or the bundle -
// the same file set the platform's compile job sees when it clones the repo
// at a commit. Outside a git repository, it falls back to a plain filesystem
// walk (listSkillFilesFS) so `validate` still works anywhere; only the
// compile-only tree-hash step (resolveSkillTreeHash) cares whether a git
// repository exists at all.
func listSkillFiles(projectDir string, skill SkillDefinition) (relPaths []string, insideRepo bool, err error) {
	root := filepath.Join(projectDir, skill.Dir)

	repoRoot, gitErr := runGit(root, "rev-parse", "--show-toplevel")
	if gitErr != nil {
		paths, walkErr := listSkillFilesFS(root)
		return paths, false, walkErr
	}

	relToRepo, err := skillDirRelToRepo(repoRoot, root)
	if err != nil {
		return nil, true, err
	}

	cmd := exec.Command("git", "ls-files", "-co", "--exclude-standard", "-z", "--", relToRepo)
	cmd.Dir = repoRoot

	out, err := cmd.Output()
	if err != nil {
		return nil, true, err
	}

	relToRepoOS := filepath.FromSlash(relToRepo)

	var paths []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" {
			continue
		}

		rel, relErr := filepath.Rel(relToRepoOS, filepath.FromSlash(p))
		if relErr != nil {
			continue
		}

		paths = append(paths, filepath.ToSlash(rel))
	}

	return paths, true, nil
}

// listSkillFilesFS walks root's filesystem tree and returns every entry's
// path relative to root, forward-slash separated. Used only as the
// not-a-git-repository fallback for listSkillFiles. Symlinks are not
// special-cased here - WalkDir does not traverse into them regardless, and
// validateSkillContent lstats every path itself to reject them.
func listSkillFilesFS(root string) ([]string, error) {
	var relPaths []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		relPaths = append(relPaths, filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))))
		return nil
	})

	return relPaths, err
}

// validateSkillContent resolves a skill's file list (listSkillFiles) and
// applies every per-file and bundle-level check ported from the server's
// skill-validator: dotfiles, size caps, path length, symlink rejection, the
// 200-file cap, and SKILL.md frontmatter. It never touches git *state* -
// tree-hash resolution is a separate, compile-only step (resolveSkillTreeHash)
// - so this is what both Load (content-only validation, used by `validate`)
// and readSkillBundle (compile) call. needAllContent additionally reads every
// file's raw bytes (not just SKILL.md's) for bundle zipping.
func validateSkillContent(projectDir string, skill SkillDefinition, needAllContent bool) ([]skillFile, []Issue) {
	root := filepath.Join(projectDir, skill.Dir)

	relPaths, _, err := listSkillFiles(projectDir, skill)
	if err != nil {
		return nil, []Issue{{File: skill.Dir, Message: fmt.Sprintf("cannot list files for skill %q: %s", skill.Slug, err.Error())}}
	}

	sort.Strings(relPaths)

	var files []skillFile
	var issues []Issue

	for _, relPath := range relPaths {
		fileID := filepath.Join(skill.Dir, filepath.FromSlash(relPath))
		absPath := filepath.Join(root, filepath.FromSlash(relPath))

		info, err := os.Lstat(absPath)
		if err != nil {
			// Tracked-but-deleted-from-disk (git ls-files -c reports index
			// entries regardless of working-tree presence); skip it.
			continue
		}

		if info.Mode()&fs.ModeSymlink != 0 {
			issues = append(issues, Issue{
				File:    fileID,
				Message: fmt.Sprintf("skill %q: %q is a symlink; symlinks are not allowed", skill.Slug, relPath),
			})
			continue
		}

		if info.IsDir() {
			continue
		}

		issues = append(issues, validateSkillFilePath(skill.Slug, fileID, relPath)...)

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
			content, err := os.ReadFile(absPath)
			if err != nil {
				issues = append(issues, Issue{File: fileID, Message: "cannot read file: " + err.Error()})
				continue
			}
			sf.content = content
		}

		files = append(files, sf)
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

	return files, issues
}

// readSkillBundle validates a skill's content and, only if its git tree hash
// resolves, returns the compiled skill: its slug and the git tree hash of its
// directory at HEAD. When skillBundlesDir is non-empty, it also writes
// <skillBundlesDir>/<treeHash>.zip with every file's raw bytes. Git state is
// advisory only for local compile (the authoritative compile job always runs
// on a clean clone where everything is committed): a genuine content problem
// (bad frontmatter, oversize file, symlink, ...) is a hard error and returns
// nil, but an unresolved tree hash (not a git repository, or the skill
// directory never committed) is reported as a warning in the returned issues
// and the skill is simply omitted (nil, warning-only issues) rather than
// failing compile.
func readSkillBundle(projectDir string, skill SkillDefinition, skillBundlesDir string) (*CompiledSkill, []Issue) {
	needAllContent := skillBundlesDir != ""

	files, issues := validateSkillContent(projectDir, skill, needAllContent)

	treeHash, resolved, hashIssues := resolveSkillTreeHash(projectDir, skill)
	issues = append(issues, hashIssues...)

	if errs, _ := PartitionIssues(issues); len(errs) > 0 {
		return nil, issues
	}

	if !resolved {
		return nil, issues // git state unresolved: warning only, skill omitted
	}

	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })

	if needAllContent {
		if err := writeSkillBundleZip(skillBundlesDir, treeHash, files); err != nil {
			return nil, append(issues, Issue{File: skill.Dir, Message: fmt.Sprintf("cannot write bundle zip for skill %q: %s", skill.Slug, err.Error())})
		}
	}

	return &CompiledSkill{Slug: skill.Slug, TreeHash: treeHash}, issues
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

// skillDirRelToRepo resolves a skill directory's path relative to a git
// repository root, evaluating symlinks first (macOS puts TMPDIR behind a
// /var -> /private/var symlink) so the two paths are comparable instead of
// producing a bogus relative path that never matches anything at HEAD. git
// itself already resolves symlinks in `rev-parse --show-toplevel`, so only
// skillDir needs the same treatment here.
func skillDirRelToRepo(repoRoot, skillDir string) (string, error) {
	realSkillDir, err := filepath.EvalSymlinks(skillDir)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(repoRoot, realSkillDir)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(rel), nil
}

// resolveSkillTreeHash computes the compiled tree hash for a skill directory:
// the git tree object hash of the directory at HEAD (git rev-parse
// HEAD:<repo-relative-path>). Git state is advisory only, never a hard error:
// local compile is a convenience ahead of the authoritative compile job,
// which always runs on a clean clone where every directory is committed, so
// nothing uncommitted locally can ever reach a deploy. If the project isn't
// inside a git repository, or the skill directory has never been committed,
// resolved is false and the returned issue is a warning telling the caller to
// skip the skill (readSkillBundle omits it from the compiled config rather
// than failing compile). If the skill directory has uncommitted changes, the
// hash still resolves (it reflects HEAD), but that's reported as a warning
// too.
func resolveSkillTreeHash(projectDir string, skill SkillDefinition) (hash string, resolved bool, issues []Issue) {
	absSkillDir := filepath.Join(projectDir, skill.Dir)

	repoRoot, err := runGit(absSkillDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", false, []Issue{{
			Severity: SeverityWarning,
			File:     skill.Dir,
			Message:  fmt.Sprintf("skill %q skipped: not inside a git repository; commit the skill directory (%s) to compile it - the tree hash is derived from git", skill.Slug, skill.Dir),
		}}
	}

	relToRepo, err := skillDirRelToRepo(repoRoot, absSkillDir)
	if err != nil {
		return "", false, []Issue{{Severity: SeverityWarning, File: skill.Dir, Message: "cannot resolve skill directory relative to the git repository root: " + err.Error()}}
	}

	headHash, err := runGit(repoRoot, "rev-parse", "HEAD:"+relToRepo)
	if err != nil || !skillTreeHashPattern.MatchString(headHash) {
		return "", false, []Issue{{
			Severity: SeverityWarning,
			File:     skill.Dir,
			Message:  fmt.Sprintf("skill %q skipped: commit the skill directory (%s) to compile it - the tree hash is derived from git", skill.Slug, relToRepo),
		}}
	}

	if status, statusErr := runGit(repoRoot, "status", "--porcelain", "--", relToRepo); statusErr == nil && status != "" {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			File:     skill.Dir,
			Message:  fmt.Sprintf("skill %q has uncommitted changes; the tree hash reflects the last commit (HEAD), not your working tree - commit before pushing", skill.Slug),
		})
	}

	return headHash, true, issues
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
