---
name: major
description: >
  Use the Major platform to create, develop, and deploy web applications.
  Triggers when user mentions Major apps, deploying, creating apps,
  managing resources, or working with the Major CLI.
disable-model-invocation: false
allowed-tools: Bash(major *)
---

# Major Platform

Major is a platform for building and deploying web applications. It creates GitHub-backed apps with local development, connected resources (databases, APIs), and production deployments.

## Command Reference

### Application Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major app create --name "X" --description "Y"` | Create a new app (skips resource selection in non-interactive mode) | Direct |
| `major app clone --app-id "UUID"` | Clone an existing app | Direct |
| `major app start` | Start local dev server | Direct |
| `major app deploy --message "description"` | Deploy to production | Direct |
| `major app list` | List all apps in org (JSON: id, name) | Direct |
| `major app info` | Show current app ID | Direct |
| `major app configure` | Open app settings in browser | Direct |

### Resource Commands

| Command | Description | Mode |
|---------|-------------|------|
| `major resource env` | View/switch environments | Direct |
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
| `major org whoami` | Show current default org | Direct |
| `major org select` | Select default organization | Interactive |

### Other Commands

| Command | Description |
|---------|-------------|
| `major update` | Update CLI to latest version |
| `major docs` | Open documentation in browser |

## Rules

**Direct** commands: Run these yourself via Bash.
**Interactive** commands: Tell the user to run these in their terminal — they require browser or TUI interaction.

### Critical Rules

1. **NEVER use raw git commands** (`git clone`, `git push`) — always use Major CLI commands. `major app clone` handles GitHub auth, permissions, and `.env` generation.

2. **Always use `--message` with deploy** to skip the interactive commit prompt. On first deploy, also pass `--slug` to set the URL non-interactively:
   ```bash
   major app deploy --message "Add search feature" --slug "my-app"
   ```

3. **Always check auth first** before running commands:
   ```bash
   major user whoami
   ```

4. **GitHub Invitation Flow** — When you see "Action Required: Accept GitHub Invitation":
   - STOP and tell the user to accept the invitation at the URL shown
   - Tell them a browser window should have opened automatically
   - After they accept, re-run the same command
   - Do NOT try `git clone` directly or retry without user action

5. **App type**: Creates a NextJS application by default

## Workflow Reference

For detailed workflows, see the docs below:

- [Getting Started](docs/getting-started.md) — Install, auth, first app
- [App Workflows](docs/app-workflows.md) — Create, clone, start, deploy
- [Resource Workflows](docs/resource-workflows.md) — Create, manage, environments
- [Org Management](docs/org-management.md) — Organizations and teams
- [Troubleshooting](docs/troubleshooting.md) — Common issues and fixes
