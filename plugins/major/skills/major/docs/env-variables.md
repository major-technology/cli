# Environment Variable Workflows

Each environment variable holds one value per app. The same value applies in every environment (development, staging, production).

## Listing Variables

```bash
major vars list
```

Outputs a table with masked values.

```bash
major vars list --show-values
```

Reveals full values.

```bash
major vars list --json
```

JSON output with full values -- suitable for scripting. Shape: `{"variables":[{"key":"...","value":"..."}]}`.

## Getting a Single Variable

```bash
major vars get DATABASE_URL
```

Prints the raw value to stdout with no prefix -- suitable for shell use:

```bash
export DB=$(major vars get DATABASE_URL)
```

Returns exit code 1 if the key does not exist.

```bash
major vars get DATABASE_URL --json
```

Wraps the result as `{"key":"...","value":"..."}`.

## Setting Variables

```bash
major vars set DATABASE_URL=postgres://localhost/mydb
```

Creates or updates the value.

The argument is split on the **first** `=`, so values can contain `=`:

```bash
major vars set 'CONNECTION_STRING=host=db;port=5432;user=app'
```

### Key Rules

- Must match `^[A-Za-z_][A-Za-z0-9_]*$`
- Cannot start with `MAJOR_` (reserved for platform-managed variables)
- Setting is idempotent -- running the same command again is a no-op

## Removing Variables

```bash
major vars unset SECRET_KEY --yes
```

Removes the key. Always pass `--yes` in automated/agentic contexts to skip the confirmation prompt.

## Pulling Variables to a Local File

```bash
major vars pull
```

Writes all variables (user-defined and platform `MAJOR_*` vars) to `.env` in dotenv format. Automatically adds `.env` to `.gitignore` if it is not already ignored.

```bash
major vars pull --file .env.local
```

Writes to a custom file.

### File Format

```bash
# Pulled from Major at 2026-04-13T10:00:00Z
# Do not edit MAJOR_* variables - they are managed by the platform

DATABASE_URL=postgres://localhost/mydb
STRIPE_SECRET_KEY="sk_test_abc123"

MAJOR_API_BASE_URL=https://api.major.build
MAJOR_JWT_TOKEN="eyJ..."
```

User-defined keys are sorted alphabetically first, followed by `MAJOR_*` system vars. Values containing special characters (`$`, `#`, spaces, quotes, newlines) are double-quoted with proper escaping.

## Common Patterns

### Set Up Local Development

```bash
major vars pull        # download vars
major app start        # start dev server
```

### Check What's Set Before Deploying

```bash
major vars list --show-values
```
