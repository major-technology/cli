# Getting Started with Major

## Prerequisites

1. **Node.js** >= 22.0.0
2. **pnpm** installed globally

## Install the Major CLI

```bash
curl -fsSL https://install.major.build | bash
```

## Authenticate

Authentication is interactive -- the user must run this in their terminal:

```bash
major user login
```

This opens a browser for OAuth authentication, then prompts for default organization selection. Credentials are stored securely in the system keychain.

## Verify Authentication

```bash
major user whoami
```

Returns the logged-in email and default organization.

## Configure GitHub Username

```bash
major user gitconfig
```

Stores the GitHub username in the keychain for app creation. This is auto-detected from SSH config in most cases.

## Create Your First App

```bash
major app create --name "my-app" --description "My first Major app"
```

This will:
1. Create the app and GitHub repository (Next.js by default)
2. Ensure GitHub repo access (may trigger invitation flow)
3. Clone the repository locally
4. Generate `.env` file with environment variables
5. Generate `.mcp.json` for Claude Code integration

## Start Development

```bash
cd my-app
major app start
```

This checks for upstream changes, runs `pnpm install` followed by `pnpm dev`, starting a local dev server with access to connected resources via environment variables.

## Deploy to Production

```bash
major app deploy --message "Initial deployment" --no-wait
```

Always include `--message` to skip the interactive commit prompt. Use `--no-wait` to skip the TUI progress tracker. The CLI stages, commits, pushes, and deploys automatically.

## Check App Status

```bash
major app info
```

Shows app ID, name, deploy status, and URL (if deployed).
