---
name: major
description: >
  Use the Major platform to manage apps, agents, skills, resources, and deployments.
  Triggers when user mentions Major apps, agents, skills, deploying,
  managing resources, or working with the Major CLI.
disable-model-invocation: false
allowed-tools: Bash(major *), Read(**/plugins/major/skills/major/docs/*)
---

# Major Platform

Major is a platform for building and deploying Next.js web applications. It creates GitHub-backed Next.js apps with local development, connected resources (databases, APIs), and production deployments.

## Command Reference

### Application Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major app create --name "X" --description "Y"` | Create a new app (skips resource selection in non-interactive mode) | Direct |
| `major app clone --app-id "UUID"` | Clone an existing app | Direct |
| `major app start` | Start local dev server (warns if behind origin) | Direct |
| `major app deploy --message "description" --no-wait` | Deploy to production (returns version ID) | Direct |
| `major app deploy-status --version-id "ID"` | Check deployment status (JSON: status, appUrl, error) | Direct |
| `major app list [--editable] [--json]` | List visible apps, including undeployed apps; optionally only editable ones | Direct |
| `major app info` | Show app ID, name, deploy status, URL | Direct |
| `major app info --json` | App info as JSON | Direct |
| `major app configure` | Open app settings in browser | Direct |
| `major app logs` | Show recent application logs (newest-first) | Direct |
| `major app logs --since 30m --search "error"` | Filter logs by time window and substring | Direct |
| `major app logs --json` | Output logs as JSON (includes `nextToken` for pagination) | Direct |
| `major app logs --preview` | Read the sandbox dev server's logs | Direct |
| `major app errors list` | List active runtime errors | Direct |
| `major app errors get <errorId>` | Inspect an error and its stack trace | Direct |
| `major app errors resolve <errorId>` | Mark a confirmed error fixed after committing the fix | Direct |
| `major app errors enable` | Enable runtime error reporting after adding its scaffolding | Direct |
| `major app theme list` | List available themes | Direct |
| `major app theme get` | Read the app's current theme | Direct |
| `major app theme apply <themeId>` | Select a theme and write its files into the checkout | Direct |
| `major app ai-proxy status` | Inspect AI proxy configuration and spend | Direct |
| `major app ai-proxy enable` | Enable the proxy with a $10/month limit (ask the user first) | Direct |

### Agent Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major agent list [--editable]` | List visible agents; optionally only editable ones | Direct |
| `major agent get [agent-id]` | Read agent detail and declared env-key status (never values) | Direct |
| `major agent create --name "X" [--description "Y"]` | Create an unpublished draft; does not mount it | Direct |
| `major agent run [agent-id] --prompt "..." [--name "title"]` | Start an independent run — **human CLI token only**; AI sessions use MCP `run_agent` | MCP for AI |
| `major agent runs list [--agent <agent-id>] [--all-users]` | List your runs; `--all-users` includes others' runs only on editable agents | Direct |
| `major agent runs content <run-id> [--limit N]` | Read run messages | Direct |
| `major agent runs send <run-id> --message "..."` | Send a follow-up message | Direct |
| `major agent runs stop <run-id>` | Stop a run | Direct |
| `major agent channel connect [agent-id] --type slack` | Connect Slack — **human CLI token only**; AI sessions use MCP `connect_agent_to_slack` | MCP for AI |
| `major agent channel pause [agent-id] --type slack` | Pause Slack replies | Direct |
| `major agent channel resume [agent-id] --type slack` | Resume Slack replies | Direct |
| `major agent channel delete [agent-id] --type slack` | Delete a Slack connection — human CLI only; agents must not do this | Human only |
| `major agent permissions resource [agent-id] <resource-id>` | Inspect published resource tool permissions | Direct |
| `major agent permissions app [agent-id] <app-id>` | Inspect published app endpoint permissions | Direct |

### Skill Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major skill list [--editable] [--published]` | List visible skills; optionally narrow to editable or published | Direct |
| `major skill get [skill-id]` | Read skill detail | Direct |
| `major skill create` | Create an unpublished draft; does not mount it | Direct |

Agent and skill target commands use the matching `.major/config.json` in a mounted workspace when the ID is omitted; an explicit ID wins. Lists and creates do not need a workspace. Use `--json` for machine-readable output.

Before mounting, use MCP `list_apps`, `list_agents`, or `list_skills` to discover targets. After mounting, use the CLI for reads and run follow-ups. To start a run in an AI session, call the approval-gated `mcp__orchestrator-platform__run_agent({agentId, prompt})`, not `major agent run`. To connect Slack, use approval-gated `mcp__orchestrator-platform__connect_agent_to_slack({agentId})`, not the CLI. Do not use the removed app-scoped `major-app` run tools. The deployed-app runtime API is separate.

### Environment Variable Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major vars list` | List env vars for current environment (masked values) | Direct |
| `major vars list --show-values` | List env vars with full values | Direct |
| `major vars list --json` | List env vars as JSON (includes full values) | Direct |
| `major vars get KEY` | Print a single env var's raw value | Direct |
| `major vars get KEY --json` | Print a single env var as JSON | Direct |
| `major vars set KEY=VALUE` | Create or update an env var | Direct |
| `major vars unset KEY` | Remove an env var from current env | Interactive |
| `major vars unset KEY --yes` | Remove an env var without prompting | Direct |
| `major vars unset KEY --all-environments --yes` | Remove an env var from all environments | Direct |
| `major vars pull` | Download env vars to local .env file | Direct |
| `major vars pull --file .env.staging` | Download to a custom file path | Direct |

Prefer entering secrets through Major's frontend; never ask the user to paste them into chat. Use `major vars set` for known values the user explicitly wants configured.

All vars commands accept `--env <name>` to target a specific environment (case-insensitive). Without it, they use the user's currently-selected environment.

### Resource Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major resource list` | List org resources as JSON (no app workspace required) | Direct |
| `major resource add --id "UUID"` | Add a resource to current app | Direct |
| `major resource remove --id "UUID"` | Remove a resource from current app | Direct |
| `major resource env` | View/switch environments (interactive, or `--id` for non-interactive) | Direct |
| `major resource env-list` | List available environments | Direct |
| `major resource env-list --json` | List environments as JSON | Direct |
| `major resource create` | Open resource creation in browser | Direct |
| `major resource manage` | Interactive resource menu | Interactive |

### User Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major user whoami` | Check authentication status | Direct |
| `major user gitconfig` | Configure GitHub username | Direct |
| `major user login` | Authenticate (opens browser) | Interactive |
| `major user logout` | Log out | Direct |

### Organization Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major org list` | List all organizations | Direct |
| `major org list --json` | List organizations as JSON (includes IDs) | Direct |
| `major org whoami` | Show current default org | Direct |
| `major org select` | Select default organization | Interactive |
| `major org select --id "UUID"` | Select organization non-interactively | Direct |

### Other Commands

| Command | Description |
|---------|-------------|
| `major update` | Update CLI to latest version |
| `major docs` | Open documentation in browser |

## Rules

**Direct** commands: Run these yourself via Bash.
**MCP for AI** commands: Use the named approval-gated MCP tool; the CLI form is for human CLI tokens only.
**Human only** commands: Do not run these as an agent.
**Interactive** commands: Tell the user to run these in their terminal -- they require browser or TUI interaction.

### Critical Rules

1. **NEVER use raw git commands** (`git clone`, `git push`) -- always use Major CLI commands. `major app clone` handles GitHub auth, permissions, and `.env` generation.

2. **Always use `--message` and `--no-wait` with deploy** to skip the interactive commit prompt and avoid TUI issues. The command returns a version ID you can use to check status:
   ```bash
   major app deploy --message "Add search feature" --no-wait
   # Returns version ID, then check status:
   major app deploy-status --version-id "<version-id>"
   ```
   On first deploy, also pass `--slug` to set the URL non-interactively:
   ```bash
   major app deploy --message "Initial deploy" --slug "my-app" --no-wait
   ```

3. **Always check auth first** before running commands:
   ```bash
   major user whoami
   ```

4. **GitHub Invitation Flow** -- When you see "Action Required: Accept GitHub Invitation":
   - STOP and tell the user to accept the invitation at the URL shown
   - Tell them a browser window should have opened automatically
   - After they accept, re-run the same command
   - Do NOT try `git clone` directly or retry without user action

5. **App type**: Creates a Next.js application by default

6. **Resource management**: Use `major resource list` to see available resources, then `major resource add --id <id>` or `major resource remove --id <id>` to manage them programmatically. Use `major resource env-list --json` to see environments and `major resource env --id <id>` to switch.

8. **Environment variable management**: Use `major vars` commands to manage env vars. Keys must match `^[A-Za-z_][A-Za-z0-9_]*$` and cannot start with `MAJOR_` (reserved for the platform). Always use `--yes` with `major vars unset` to avoid interactive prompts. Use `major vars pull` to sync env vars to a local `.env` file -- it auto-updates `.gitignore`.

7. **Organization selection**: Use `major org list --json` to get org IDs, then `major org select --id <id>` to switch orgs programmatically.

## Workflow Reference

For detailed workflows, see the docs below:

- [Getting Started](docs/getting-started.md) -- Install, auth, first app
- [App Workflows](docs/app-workflows.md) -- Create, clone, start, deploy
- [Env Variables](docs/env-variables.md) -- Set, list, pull, and manage env vars per environment
- [Resource Workflows](docs/resource-workflows.md) -- Create, manage, environments
- [Org Management](docs/org-management.md) -- Organizations and teams
- [Troubleshooting](docs/troubleshooting.md) -- Common issues and fixes
