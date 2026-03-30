# Major Claude Code Plugin

Use the Major platform agentically through Claude Code. Create, develop, and deploy web apps without leaving your terminal.

## Install

### From Marketplace

```
/plugin marketplace add major-technology/cli
/plugin install major
```

### Local Development

```bash
claude --plugin-dir ./claude-code-plugin
```

Use `/reload-plugins` to pick up changes without restarting.

## What It Does

This plugin teaches Claude Code how to use the Major CLI. Once installed, Claude can:

- Create and clone Major apps
- Start local dev servers
- Deploy to production
- Manage resources and environments
- Handle GitHub invitation flows
- Troubleshoot common issues

## Usage

Claude will automatically use the Major skill when you mention Major-related tasks. You can also invoke it directly:

```
/major
```

## Plugin Structure

```
claude-code-plugin/
├── .claude-plugin/
│   ├── plugin.json          # Plugin manifest
│   └── marketplace.json     # Distribution manifest
├── skills/
│   └── major/
│       ├── SKILL.md         # Main skill
│       └── docs/            # Workflow reference
│           ├── getting-started.md
│           ├── app-workflows.md
│           ├── resource-workflows.md
│           ├── org-management.md
│           └── troubleshooting.md
└── README.md
```

## Replaces

This plugin replaces the MCP server at `mono-builder/apps/mcp-server/`. All documentation, workflow prompts, and system instructions have been migrated to the skill and docs structure.
