package projects

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// copyFixtureDir recursively copies src into dst, used to stage a testdata
// fixture (not itself a git repo) into a throwaway one.
func copyFixtureDir(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, content, 0644)
	})
	if err != nil {
		t.Fatalf("copy fixture %s -> %s: %v", src, dst, err)
	}
}

// runTestGit runs git in dir, failing the test on error.
func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// initGitRepo git-inits dir and commits everything in it, so HEAD:<path>
// tree-hash lookups resolve the way they would in a real project.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	runTestGit(t, dir, "init")
	runTestGit(t, dir, "config", "user.email", "test@example.com")
	runTestGit(t, dir, "config", "user.name", "Test")
	runTestGit(t, dir, "add", "-A")
	runTestGit(t, dir, "commit", "-m", "fixture")
}

func TestCompileProjectWithSkills(t *testing.T) {
	dir := t.TempDir()
	copyFixtureDir(t, "testdata/with-skills", dir)

	// A binary file (non-UTF8 bytes) proves bundle files are never decoded as
	// text and round-trip byte-identical through the zip.
	binPath := filepath.Join(dir, "src", "skills", "pdf-tools", "assets", "icon.bin")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatal(err)
	}
	binContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}
	if err := os.WriteFile(binPath, binContent, 0644); err != nil {
		t.Fatal(err)
	}

	initGitRepo(t, dir)

	bundlesDir := t.TempDir()
	res, issues := CompileWithSkillBundles(dir, bundlesDir)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	if len(res.Config.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(res.Config.Skills))
	}

	skill := res.Config.Skills[0]
	if skill.Slug != "pdf-tools" {
		t.Fatalf("skill slug = %q, want pdf-tools", skill.Slug)
	}
	if !skillTreeHashPattern.MatchString(skill.TreeHash) {
		t.Fatalf("tree hash %q does not look like a git tree hash", skill.TreeHash)
	}
	if skill.Bundle != nil {
		t.Fatalf("CLI compile must never set bundle, got: %+v", skill.Bundle)
	}

	// No file contents anywhere in the compiled config.
	if strings.Contains(string(res.ConfigJSON), "console.log") {
		t.Fatalf("compiled config must not inline file contents: %s", res.ConfigJSON)
	}

	zipPath := filepath.Join(bundlesDir, skill.TreeHash+".zip")
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open bundle zip: %v", err)
	}
	defer zr.Close()

	got := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %s: %v", f.Name, err)
		}
		got[f.Name] = content
	}

	for _, name := range []string{"SKILL.md", "references/usage.md", "scripts/run.js", "assets/icon.bin"} {
		if _, ok := got[name]; !ok {
			t.Fatalf("expected %s in bundle zip, got: %v", name, got)
		}
	}

	if !bytes.Equal(got["assets/icon.bin"], binContent) {
		t.Fatalf("binary file not byte-identical: got %v want %v", got["assets/icon.bin"], binContent)
	}

	origSkillMD, err := os.ReadFile(filepath.Join(dir, "src", "skills", "pdf-tools", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got["SKILL.md"], origSkillMD) {
		t.Fatalf("SKILL.md not byte-identical")
	}

	if len(res.Config.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(res.Config.Agents))
	}
}

func TestCompileWithoutBundlesDirWritesNoZips(t *testing.T) {
	dir := t.TempDir()
	copyFixtureDir(t, "testdata/with-skills", dir)
	initGitRepo(t, dir)

	res, issues := Compile(dir)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	if len(res.Config.Skills) != 1 || res.Config.Skills[0].TreeHash == "" {
		t.Fatalf("expected 1 skill with a tree hash, got: %+v", res.Config.Skills)
	}
}

func TestCompileNoSkillsDirOmitsSkillsKey(t *testing.T) {
	res, issues := Compile("testdata/valid")
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}

	if len(res.Config.Skills) != 0 {
		t.Fatalf("expected no skills, got: %#v", res.Config.Skills)
	}

	if strings.Contains(string(res.ConfigJSON), `"skills"`) {
		t.Fatalf("expected no skills key in compiled JSON, got: %s", res.ConfigJSON)
	}
}

func TestCompileInvalidSkillSlugRejected(t *testing.T) {
	_, issues := Compile("testdata/invalid-skill-slug")
	if !findIssue(issues, "invalid skill slug") {
		t.Fatalf("expected invalid-slug issue, got: %+v", issues)
	}
}

func TestCompileSkillBadFrontmatterRejected(t *testing.T) {
	_, issues := Compile("testdata/skill-bad-frontmatter")
	if !findIssue(issues, "frontmatter block") {
		t.Fatalf("expected frontmatter issue, got: %+v", issues)
	}
}

// TestCompileDuplicateSkillSlugSkipped documents that two skills can't
// collide on slug via real directory scanning: the slug IS the directory
// name, and a filesystem cannot hold two entries of the same name in one
// directory. discoverSkills still guards against it defensively (skills.go),
// but there is no way to construct this on disk to exercise that path.
func TestCompileDuplicateSkillSlugSkipped(t *testing.T) {
	t.Skip("duplicate skill slugs cannot be constructed on disk: slug == directory name")
}

func TestCompileSkillSymlinkRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "linked")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Linked\ndescription: has a symlink\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "real.md")
	if err := os.WriteFile(target, []byte("real"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(skillDir, "linked.md")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	_, issues := Compile(dir)
	if !findIssue(issues, "is a symlink") {
		t.Fatalf("expected symlink issue, got: %+v", issues)
	}
}

func TestCompileSkillOversizeFileRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "big")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Big\ndescription: has an oversize file\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	big := make([]byte, MaxSkillFileBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(skillDir, "notes.txt"), big, 0644); err != nil {
		t.Fatal(err)
	}

	_, issues := Compile(dir)
	if !findIssue(issues, "per-file size cap") {
		t.Fatalf("expected oversize-file issue, got: %+v", issues)
	}
}

// TestCompileSkillArbitraryExtensionAllowed checks that the old server-side
// extension allow-list is gone: binary asset types like .png/.woff2 compile
// cleanly now.
func TestCompileSkillArbitraryExtensionAllowed(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "media")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Media\ndescription: Has binary assets.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "logo.png"), []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x01}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "font.woff2"), []byte{0x77, 0x4F, 0x46, 0x32, 0x00, 0xFF, 0x10}, 0644); err != nil {
		t.Fatal(err)
	}

	initGitRepo(t, dir)

	_, issues := Compile(dir)
	if len(issues) != 0 {
		t.Fatalf("expected .png/.woff2 files to be allowed now, got issues: %+v", issues)
	}
}

func TestCompileSkillFileCountCapRaisedTo200(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "many")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Many\ndescription: has many files.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// SKILL.md plus MaxSkillFiles more is one over the cap.
	for i := 0; i < MaxSkillFiles; i++ {
		name := filepath.Join(skillDir, fmt.Sprintf("file-%d.txt", i))
		if err := os.WriteFile(name, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	initGitRepo(t, dir)

	_, issues := Compile(dir)
	if !findIssue(issues, fmt.Sprintf("max file count of %d", MaxSkillFiles)) {
		t.Fatalf("expected file-count cap issue, got: %+v", issues)
	}
}

// TestCompileSkillNotGitRepoSkipsWithWarning checks that compiling outside a
// git repository never fails: a skill whose tree hash can't be resolved is
// reported as a warning and simply omitted from the compiled config. Local
// compile is advisory - the authoritative compile job always runs on a clean
// clone where everything is committed - so nothing about git state may block
// it.
func TestCompileSkillNotGitRepoSkipsWithWarning(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "orphan")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Orphan\ndescription: never committed.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	res, issues := Compile(dir)
	if res == nil {
		t.Fatalf("expected compile to succeed despite the skipped skill, got nil result with issues: %+v", issues)
	}
	if len(res.Config.Skills) != 0 {
		t.Fatalf("expected the unresolved skill to be omitted, got: %+v", res.Config.Skills)
	}
	if strings.Contains(string(res.ConfigJSON), `"skills"`) {
		t.Fatalf("expected no skills key when the only skill is skipped, got: %s", res.ConfigJSON)
	}
	if !findIssue(issues, "not inside a git repository") {
		t.Fatalf("expected a not-a-git-repository warning, got: %+v", issues)
	}
	if errs, _ := PartitionIssues(issues); len(errs) != 0 {
		t.Fatalf("expected no error-severity issues, got: %+v", errs)
	}
}

// TestCompileSkillNeverCommittedSkipsWithWarning mirrors
// TestCompileSkillNotGitRepoSkipsWithWarning for a skill directory that is
// inside a git repository but was never committed.
func TestCompileSkillNeverCommittedSkipsWithWarning(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	initGitRepo(t, dir) // commits project.json but not the skill created below

	skillDir := filepath.Join(dir, "src", "skills", "wip")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: WIP\ndescription: never committed.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	res, issues := Compile(dir)
	if res == nil {
		t.Fatalf("expected compile to succeed despite the skipped skill, got nil result with issues: %+v", issues)
	}
	if len(res.Config.Skills) != 0 {
		t.Fatalf("expected the never-committed skill to be omitted, got: %+v", res.Config.Skills)
	}
	if !findIssue(issues, "commit the skill directory") {
		t.Fatalf("expected a never-committed warning, got: %+v", issues)
	}
	if errs, _ := PartitionIssues(issues); len(errs) != 0 {
		t.Fatalf("expected no error-severity issues, got: %+v", errs)
	}
}

// TestCompileSkillUncommittedChangesWarns checks that a committed-then-locally
// -modified skill directory still compiles (the tree hash reflects HEAD), but
// reports the discrepancy as a warning, not an error.
func TestCompileSkillUncommittedChangesWarns(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "dirty")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	skillMD := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMD, []byte("---\nname: Dirty\ndescription: has local edits.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	initGitRepo(t, dir)

	// Edit after committing: HEAD has the original content, the working tree
	// does not - the compiled hash must still resolve (it reflects HEAD), but
	// this must be reported as a warning.
	if err := os.WriteFile(skillMD, []byte("---\nname: Dirty\ndescription: has local edits.\n---\nBody, edited.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	res, issues := Compile(dir)
	if res == nil {
		t.Fatalf("expected compile to succeed with only a warning, got nil result with issues: %+v", issues)
	}
	if len(res.Config.Skills) != 1 || res.Config.Skills[0].TreeHash == "" {
		t.Fatalf("expected the dirty skill to still compile with a tree hash, got: %+v", res.Config.Skills)
	}
	if !findIssue(issues, "uncommitted changes") {
		t.Fatalf("expected an uncommitted-changes warning, got: %+v", issues)
	}
	for _, issue := range issues {
		if !issue.IsWarning() {
			t.Fatalf("expected every issue to be a warning, got an error: %+v", issue)
		}
	}
}

// TestValidateGitignoredDotDSStoreClean checks that a gitignored .DS_Store
// inside a skill directory is invisible to validate: it is excluded by the
// git-lens file listing (listSkillFiles), so it never reaches the dotfile
// check at all. The artifact that ships is the git tree the platform compile
// job clones at a commit, so a gitignored file is phantom content.
func TestValidateGitignoredDotDSStoreClean(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "clean")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Clean\ndescription: has a gitignored .DS_Store.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".DS_Store\n"), 0644); err != nil {
		t.Fatal(err)
	}

	initGitRepo(t, dir) // commits project.json, the skill, and .gitignore

	// Created after the commit, and ignored: never enters the index.
	if err := os.WriteFile(filepath.Join(skillDir, ".DS_Store"), []byte("junk"), 0644); err != nil {
		t.Fatal(err)
	}

	if issues := Validate(dir); len(issues) != 0 {
		t.Fatalf("expected no issues for a gitignored .DS_Store, got: %+v", issues)
	}

	bundlesDir := t.TempDir()
	res, issues := CompileWithSkillBundles(dir, bundlesDir)
	if len(issues) != 0 {
		t.Fatalf("expected no compile issues, got: %+v", issues)
	}

	zipPath := filepath.Join(bundlesDir, res.Config.Skills[0].TreeHash+".zip")
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open bundle zip: %v", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name == ".DS_Store" {
			t.Fatalf("expected .DS_Store to be excluded from the bundle zip, got entries: %v", zr.File)
		}
	}
}

// TestValidateNonIgnoredDotDSStoreRejected checks that a .DS_Store that is
// NOT gitignored still trips the dotfile rule: the server rejects it at
// publish, so a tracked or untracked-but-not-ignored dotfile must still be a
// validation error.
func TestValidateNonIgnoredDotDSStoreRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "unclean")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Unclean\ndescription: has a tracked .DS_Store.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, ".DS_Store"), []byte("junk"), 0644); err != nil {
		t.Fatal(err)
	}

	initGitRepo(t, dir) // commits the .DS_Store too - it is tracked, not ignored

	if !findIssue(Validate(dir), "dotfile") {
		t.Fatalf("expected a dotfile issue for a non-ignored .DS_Store, got: %+v", Validate(dir))
	}
}

// TestValidateNeverCommittedSkillHasNoGitIssues checks that validate never
// mentions git state: a skill directory that has never been committed
// validates its content (a bad frontmatter is still reported) without any
// "not a git repository" / "commit" noise.
func TestValidateNeverCommittedSkillHasNoGitIssues(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)
	initGitRepo(t, dir) // commits project.json but not the skill created below

	skillDir := filepath.Join(dir, "src", "skills", "wip")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Missing "description" in frontmatter: a genuine content problem that
	// must still surface.
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: WIP\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	issues := Validate(dir)
	if !findIssue(issues, "description") {
		t.Fatalf("expected the bad-frontmatter issue to still be reported, got: %+v", issues)
	}
	if findIssue(issues, "git") || findIssue(issues, "commit") {
		t.Fatalf("expected no git-related issues from validate, got: %+v", issues)
	}
}

// TestValidateOutsideGitRepoFallsBackToFilesystemWalk checks that validate
// works in a directory that is not a git repository at all: listSkillFiles
// falls back to a plain filesystem walk.
func TestValidateOutsideGitRepoFallsBackToFilesystemWalk(t *testing.T) {
	dir := t.TempDir()
	writeFixtureProject(t, dir)

	skillDir := filepath.Join(dir, "src", "skills", "no-repo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: No Repo\ndescription: not a git repo at all.\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if issues := Validate(dir); len(issues) != 0 {
		t.Fatalf("expected no issues validating outside a git repository, got: %+v", issues)
	}
}

// TestPartitionIssuesOnlyWarningsMeansSuccess exercises the seam cmd/project
// uses to decide exit codes: a run with only warning-severity issues has zero
// errors, which the CLI treats as success (exit 0).
func TestPartitionIssuesOnlyWarningsMeansSuccess(t *testing.T) {
	issues := []Issue{
		{File: "src/skills/dirty", Message: "has uncommitted changes", Severity: SeverityWarning},
		{File: "src/skills/dirty", Message: "also skipped", Severity: SeverityWarning},
	}

	errs, warnings := PartitionIssues(issues)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %+v", errs)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got: %+v", warnings)
	}

	mixed := append(issues, Issue{File: "project.json", Message: "missing name"})
	errs, _ = PartitionIssues(mixed)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error in a mixed set, got: %+v", errs)
	}
}
