---
name: major
description: >
  Use the Major platform to create, develop, and deploy Next.js web applications.
  Triggers when user mentions Major apps, deploying, creating apps,
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
| `major app list` | List all apps in org (JSON: id, name) | Direct |
| `major app info` | Show app ID, name, deploy status, URL | Direct |
| `major app info --json` | App info as JSON | Direct |
| `major app configure` | Open app settings in browser | Direct |
| `major app logs` | Show recent application logs (newest-first) | Direct |
| `major app logs --since 30m --search "error"` | Filter logs by time window and substring | Direct |
| `major app logs --json` | Output logs as JSON (includes `nextToken` for pagination) | Direct |

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

All vars commands accept `--env <name>` to target a specific environment (case-insensitive). Without it, they use the user's currently-selected environment.

### Resource Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major resource list` | List org resources as JSON (shows which are attached to app) | Direct |
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
