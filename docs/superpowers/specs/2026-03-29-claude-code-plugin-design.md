# Major Claude Code Plugin — Design Spec

**Date:** 2026-03-29
**Status:** Approved
**Goal:** Replace the MCP server at `mono-builder/apps/mcp-server/` with a Claude Code plugin that lives in the CLI repo, enabling users to use Major's full functionality agentically via Claude Code.

---

## Overview

A Claude Code plugin distributed via marketplace that teaches Claude how to use the Major CLI. Skills-first approach — no MCP server, no custom tools. Claude runs `major` commands directly via Bash.

The plugin ships a single skill (`major`) with a `docs/` subdirectory containing workflow-specific reference material. The main `SKILL.md` references these docs so Claude can pull in context as needed.

---

## Plugin Location

```
cli/claude-code-plugin/
```

Lives inside the CLI repo (`major-technology/cli`) alongside the CLI source code.

---

## Directory Structure

```
claude-code-plugin/
├── .claude-plugin/
│   ├── plugin.json              # Plugin manifest
│   └── marketplace.json         # Marketplace distribution manifest
├── skills/
│   └── major/
│       ├── SKILL.md             # Main skill — platform overview, command ref, rules
│       └── docs/
│           ├── getting-started.md    # Auth, install, first app
│           ├── app-workflows.md      # Create, clone, start, deploy, info, configure
│           ├── resource-workflows.md # Create, manage, env switching
│           ├── org-management.md     # Org select, list, whoami
│           └── troubleshooting.md    # Common issues & fixes
└── README.md
```

---

## Plugin Manifest (`plugin.json`)

```json
{
  "name": "major",
  "description": "Use the Major platform agentically — create, develop, and deploy web apps via Claude Code",
  "version": "1.0.0",
  "author": {
    "name": "Major Technology"
  },
  "repository": "https://github.com/major-technology/cli",
  "license": "MIT",
  "skills": "./skills/"
}
```

---

## Marketplace Manifest (`marketplace.json`)

```json
{
  "name": "major-tools",
  "owner": {
    "name": "Major Technology",
    "email": "support@major.build"
  },
  "metadata": {
    "description": "Official Major platform plugins for Claude Code",
    "version": "1.0.0"
  },
  "plugins": [
    {
      "name": "major",
      "source": ".",
      "description": "Use the Major platform agentically — create, develop, and deploy web apps via Claude Code",
      "version": "1.0.0",
      "author": {
        "name": "Major Technology"
      },
      "repository": "https://github.com/major-technology/cli",
      "license": "MIT"
    }
  ]
}
```

**User installation:**
1. `/plugin marketplace add major-technology/cli`
2. `/plugin install major`

---

## Main Skill (`SKILL.md`)

### Frontmatter

```yaml
---
name: major
description: >
  Use the Major platform to create, develop, and deploy web applications.
  Triggers when user mentions Major apps, deploying, creating apps,
  managing resources, or working with the Major CLI.
disable-model-invocation: false
allowed-tools: Bash(major *)
---
```

### Body Content

The skill body includes:

1. **Platform overview** — What Major does (create web apps with GitHub repos, local dev with resources, deploy to production)
2. **Command reference table** — All commands with brief descriptions, grouped by category
3. **Interactive vs non-interactive commands** — Which commands Claude can run directly vs which require user interaction
   - Direct: `user whoami`, `app create --name X --description Y --template Z`, `app clone --app-id X`, `app info`, `app start`, `app deploy --message X`, `resource env`
   - Interactive (user must run): `user login`, `resource manage`
4. **Critical rules**
   - NEVER use raw `git clone` — always `major app clone`
   - Always use `--message` flag with `major app deploy` to skip interactive prompt
   - Handle GitHub invitation flow: stop and tell user to accept invitation URL
5. **Doc references** — Links to each file in `docs/` for detailed workflow guidance

---

## Docs Files

### `getting-started.md`
- Prerequisites (Node.js, pnpm, Major CLI installed)
- Authentication flow (`major user login` — interactive, user must run)
- Checking auth status (`major user whoami`)
- Creating first app walkthrough
- GitHub username setup (`major user gitconfig`)

### `app-workflows.md`
- **Create:** `major app create --name "X" --description "Y" --template "Vite|NextJS"` — includes template options, GitHub invitation handling, resource selection
- **Clone:** `major app clone --app-id "UUID"` — never use raw git clone, handles auth + .env generation
- **Start:** `major app start` — runs pnpm install + pnpm dev
- **Deploy:** `major app deploy --message "description"` — commits, pushes, deploys, URL slug selection on first deploy
- **Info:** `major app info` — shows application ID
- **Configure:** `major app configure` — opens settings in browser

### `resource-workflows.md`
- **Create:** `major resource create` — opens browser for resource creation
- **Manage:** `major resource manage` — interactive resource menu (user must run)
- **Environment switching:** `major resource env` — view/switch between environments
- How resources connect to apps (env vars, .env file)

### `org-management.md`
- **Select default org:** `major org select` — interactive picker
- **Current org:** `major org whoami`
- **List orgs:** `major org list` — shows all with default marked

### `troubleshooting.md`
- Auth issues (token expired, re-login flow)
- App issues (missing .env, wrong directory)
- Resource issues (env vars not loading)
- Deploy issues (build failures, checking status)
- CLI issues (update command, version checking)
- GitHub invitation flow (accept + retry)

---

## Content Migration from MCP Server

The existing MCP server (`mono-builder/apps/mcp-server/`) provides:

| MCP Server Component | Migrates To |
|---|---|
| `major://docs/getting-started` resource | `docs/getting-started.md` |
| `major://docs/cli-commands` resource | Command ref table in `SKILL.md` |
| `major://docs/workflows` resource | `docs/app-workflows.md` + `docs/resource-workflows.md` |
| `major://docs/resources` resource | `docs/resource-workflows.md` |
| `major-development` prompt | Covered by `SKILL.md` rules + `docs/app-workflows.md` |
| `major-deployment` prompt | `docs/app-workflows.md` (deploy section) |
| `major-troubleshooting` prompt | `docs/troubleshooting.md` |
| System instructions | `SKILL.md` body (rules, command ref, interactive vs direct) |

After migration, the MCP server at `mono-builder/apps/mcp-server/` can be deprecated.

---

## What's NOT In Scope (v1)

- No MCP server or custom tools — CLI via Bash only
- No hooks or automation
- No custom agents
- No settings.json overrides
- No `major demo` or `major install` commands (hidden/internal)

These can be added in future versions as skills or additional plugin components.

---

## Testing

- Install locally: `claude --plugin-dir ./claude-code-plugin`
- Verify `/major` slash command appears
- Test that Claude can run Major CLI commands via the skill
- Test that docs are referenced correctly
- Use `/reload-plugins` to pick up changes during development

---

## Distribution

1. Plugin lives at `cli/claude-code-plugin/`
2. Marketplace manifest at `cli/claude-code-plugin/.claude-plugin/marketplace.json`
3. Users add marketplace: `/plugin marketplace add major-technology/cli`
4. Users install: `/plugin install major`
5. Future: submit to official Anthropic marketplace at `platform.claude.com/plugins/submit`
