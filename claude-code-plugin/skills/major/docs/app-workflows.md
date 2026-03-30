# App Workflows

## Creating an App

```bash
major app create --name "app-name" --description "What this app does"
```

**Flags:**
- `--name` — App name (required for non-interactive use)
- `--description` — App description (required for non-interactive use)

**What happens:**
1. API creates the app and a GitHub repository
2. CLI checks GitHub repo access — may trigger invitation flow
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
1. Generates/refreshes the `.env` file
2. Runs `pnpm install`
3. Runs `pnpm dev`
4. Streams output to the terminal

The dev server has access to connected resources via environment variables.

## Deploying to Production

```bash
major app deploy --message "Add search feature"
```

Always use `--message` (or `-m`) to skip the interactive commit prompt. On first deploy, also pass `--slug` to set the URL non-interactively:

```bash
major app deploy --message "Initial deployment" --slug "my-app"
```

**What happens:**
1. Checks for uncommitted git changes
2. Stages, commits, and pushes to main branch
3. On first deploy, uses `--slug` or prompts for URL slug
4. Creates application version via API
5. Shows deployment progress: bundling → building → deploying → deployed
6. Returns the live app URL on success

## Checking App Info

```bash
major app info
```

Displays the Application ID of the current directory. Must be run from within an app directory.

## Configuring an App

```bash
major app configure
```

Opens the app's settings page in the browser.
