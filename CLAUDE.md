# CLAUDE.md

## Plugin versions

The `major` plugin in `plugins/major-build` ships to Claude Code (via `.claude-plugin/marketplace.json`) and to ChatGPT/Codex (via `plugins/major-build/plugin.json`). Both hosts cache an installed plugin by its `version`, so **users only get a change once the version is bumped**. Merging a skill change without a bump leaves everyone who already installed the plugin on the old copy.

When a PR changes anything under `plugins/` (skills, references, `.mcp.json`, manifests), bump the version in all three places, to the same value:

- `plugins/major-build/.claude-plugin/plugin.json` → `version` (Claude Code; this one wins over the marketplace entry)
- `.claude-plugin/marketplace.json` → `plugins[0].version` (keep in sync with the above)
- `plugins/major-build/plugin.json` → `version` (ChatGPT/Codex)

Use semver: patch for wording/reference fixes, minor for new skills, references or connectors, major for breaking changes to tool names or MCP config. `metadata.version` in `marketplace.json` is the marketplace's own version and doesn't need to change.

After merge, users pick up the new version with `/plugin marketplace update major` in Claude Code (or automatically if auto-update is on) and `codex plugin marketplace upgrade` in Codex.
