# App Workflows

## Creating an App

```bash
major app create --name "app-name" --description "What this app does"
```

**Flags:**
- `--name` -- App name (required for non-interactive use)
- `--description` -- App description (required for non-interactive use)

**What happens:**
1. API creates the app and a GitHub repository
2. CLI checks GitHub repo access -- may trigger invitation flow
3. Prompts to select resources for the app
4. Clones the repository locally
5. Adds selected resources via major-client
6. Generates `.env` and `.mcp.json` files

### GitHub Invitation Handling

If you see this output:
```
Action Required: Accept GitHub Invitation
https://github.com/Major-Build-Apps/...
After accepting, run this command again.
```

**You must:**
1. Tell the user to accept the invitation at the URL shown
2. Tell them a browser window should have opened
3. After acceptance, re-run the same command

**Never** try `git clone` directly or retry without user action.

## Cloning an Existing App

```bash
major app clone --app-id "c21a6147-507a-4b5e-864e-f71c5996cd34"
```

Without `--app-id`, the CLI shows an interactive app picker.

**CRITICAL:** Never use `git clone` directly. `major app clone` handles GitHub authentication, permissions, and generates the required `.env` file.

## Starting Local Development

```bash
major app start
```

Must be run from the app directory. This:
1. Checks if your branch is behind `origin/main` (warns if so)
2. Generates/refreshes the `.env` file
3. Runs `pnpm install`
4. Runs `pnpm dev`
5. Streams output to the terminal

The dev server has access to connected resources via environment variables.

## Deploying to Production

```bash
major app deploy --message "Add search feature" --no-wait
```

Always use `--message` (or `-m`) to skip the interactive commit prompt. Use `--no-wait` to skip the TUI deployment progress tracker (recommended for AI-driven deploys). On first deploy, also pass `--slug` to set the URL non-interactively:

```bash
major app deploy --message "Initial deployment" --slug "my-app" --no-wait
```

**Flags:**
- `--message` / `-m` -- Commit message (skips interactive prompt)
- `--slug` -- URL slug for first deploy (skips interactive prompt)
- `--no-wait` -- Return immediately after triggering deploy (don't wait for completion)

**What happens:**
1. Checks for uncommitted git changes
2. Stages, commits, and pushes to main branch
3. On first deploy, uses `--slug` or prompts for URL slug
4. Creates application version via API
5. Without `--no-wait`: shows deployment progress (bundling -> building -> deploying -> deployed)
6. Returns the live app URL on success

## Checking App Info

```bash
major app info
```

Displays application ID, name, deploy status, and URL (if deployed). Must be run from within an app directory.

```bash
major app info --json
```

Returns the same info as JSON for programmatic use.

## Configuring an App

```bash
major app configure
```

Opens the app's settings page in the browser.
