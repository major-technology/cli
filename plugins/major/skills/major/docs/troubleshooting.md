# Troubleshooting

## Authentication Issues

**"Not authenticated" or login errors**
```bash
major user login
```
User must run this in their terminal -- it opens a browser for OAuth.

**Session expired**
Run `major user login` again to refresh the token.

**Check current auth status**
```bash
major user whoami
```

## Application Issues

**"Not in an application directory"**
Navigate to the app directory. Verify with `major app info`.

**Can't find my app**
Use `major app clone` to list and clone available apps.

**Missing `.env` file**
Run `major app start` -- it regenerates the `.env` file automatically.

**Branch behind origin**
`major app start` warns if your branch is behind `origin/main`. Run `git pull` to update.

## Resource Issues

**Resources not available locally**
1. Check connected resources: `major resource list`
2. Verify environment: `major resource env-list`
3. Restart dev server: `major app start` (regenerates `.env`)

**Wrong environment**
```bash
major resource env-list --json
major resource env --id "correct-env-id"
```
Switch to the correct environment, then restart the dev server.

**Adding/removing resources programmatically**
```bash
major resource list                    # See available resources and their IDs
major resource add --id "resource-id"  # Add to current app
major resource remove --id "resource-id"  # Remove from current app
```

## Deployment Issues

**Deploy crashes or hangs**
Use `--no-wait` to skip the TUI progress tracker:
```bash
major app deploy --message "description" --no-wait
```

**Deploy fails**
1. Check authentication: `major user whoami`
2. Verify app directory: `major app info`
3. Test locally first: `major app start`
4. Check build output for syntax or import errors

**"Not in a git repository"**
Navigate to the app directory. Major apps are git repositories.

## Organization Issues

**Switch organizations programmatically**
```bash
major org list --json                 # Get org IDs
major org select --id "org-id"        # Switch to a specific org
```

## CLI Issues

**CLI out of date**
```bash
major update
```
Auto-detects install method (brew or direct) and updates.

**Command not found**
Reinstall the CLI:
```bash
curl -fsSL https://install.major.build | bash
```

## GitHub Issues

**Invitation not accepted**
When you see "Action Required: Accept GitHub Invitation":
1. Open the URL shown in the output
2. Accept the invitation on GitHub
3. Re-run the original command

**Permission denied on clone**
Never use `git clone` directly. Use `major app clone --app-id "UUID"` which handles authentication and permissions.

## Getting More Help

```bash
major docs        # Opens documentation in browser
major --help      # Show all commands
major app --help  # Show app subcommands
```
