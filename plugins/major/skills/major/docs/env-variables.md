# Environment Variable Workflows

Environment variables are scoped per-environment (development, staging, production, etc.). Each app can have different values for the same key in different environments.

## Listing Variables

```bash
major vars list
```

Outputs a table with masked values. First line always shows the active environment.

```bash
major vars list --show-values
```

Reveals full values.

```bash
major vars list --json
```

JSON output with full values -- suitable for scripting.

### Targeting a Specific Environment

All vars commands accept `--env <name>` (case-insensitive):

```bash
major vars list --env staging
major vars list --env Production
```

Without `--env`, commands use the user's currently-selected environment (set via `major resource env`).

## Getting a Single Variable

```bash
major vars get DATABASE_URL
```

Prints the raw value to stdout with no prefix -- suitable for shell use:

```bash
export DB=$(major vars get DATABASE_URL)
```

Returns exit code 1 if the key does not exist or has no value in the target environment.

```bash
major vars get DATABASE_URL --json
```

Wraps the result as `{"key":"...","value":"...","environment":"..."}`.

## Setting Variables

```bash
major vars set DATABASE_URL=postgres://localhost/mydb
```

Creates or updates the value for the current environment only. Other environments are not affected.

The argument is split on the **first** `=`, so values can contain `=`:

```bash
major vars set 'CONNECTION_STRING=host=db;port=5432;user=app'
```

### Key Rules

- Must match `^[A-Za-z_][A-Za-z0-9_]*$`
- Cannot start with `MAJOR_` (reserved for platform-managed variables)
- Setting is idempotent -- running the same command again is a no-op

### Setting for a Specific Environment

```bash
major vars set DATABASE_URL=postgres://staging-db/app --env staging
```

## Removing Variables

```bash
major vars unset SECRET_KEY --yes
```

Removes the value for the current environment only. The key still exists if other environments have values.

```bash
major vars unset SECRET_KEY --all-environments --yes
```

Removes the key across every environment.

Always pass `--yes` in automated/agentic contexts to skip the confirmation prompt.

## Pulling Variables to a Local File

```bash
major vars pull
```

Writes all variables (user-defined and platform `MAJOR_*` vars) to `.env` in dotenv format. Automatically adds `.env` to `.gitignore` if it is not already ignored.

```bash
major vars pull --file .env.staging --env staging
```

Writes to a custom file and targets a specific environment. Note: `--env` with pull temporarily switches your active environment to fetch the correct values.

### File Format

```bash
# Pulled from Major "development" environment at 2026-04-13T10:00:00Z
# Do not edit MAJOR_* variables - they are managed by the platform

DATABASE_URL=postgres://localhost/mydb
STRIPE_SECRET_KEY="sk_test_abc123"

MAJOR_API_BASE_URL=https://api.major.build
MAJOR_JWT_TOKEN="eyJ..."
```

User-defined keys are sorted alphabetically first, followed by `MAJOR_*` system vars. Values containing special characters (`$`, `#`, spaces, quotes, newlines) are double-quoted with proper escaping.

## Listing Available Environments

Before targeting a specific environment with `--env`, you can see what's available:

```bash
major resource env-list
```

Or as JSON (useful for scripting):

```bash
major resource env-list --json
```

To switch your active environment interactively:

```bash
major resource env
```

Or non-interactively by ID:

```bash
major resource env --id "<environment-uuid>"
```

## Common Patterns

### Set Up a New Environment Locally

```bash
major resource env --id "<staging-env-id>"   # switch to staging
major vars pull                               # download vars
major app start                               # start dev server
```

### Copy a Variable Across Environments

```bash
VALUE=$(major vars get API_KEY --env production)
major vars set "API_KEY=$VALUE" --env staging
```

### Check What's Set Before Deploying

```bash
major vars list --env production --show-values
```
