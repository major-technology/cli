# Resource Workflows

Resources are external services (databases, APIs, storage) that can be connected to Major applications. They are managed at the organization level and shared across apps.

## What Are Resources?

- **Databases**: PostgreSQL, MongoDB, CosmosDB, etc.
- **APIs**: Internal company APIs, third-party services
- **Storage**: Blob storage, file systems
- **Other**: Message queues, caches, etc.

## Creating a Resource

```bash
major resource create
```

Opens the resource creation page in the browser. Resources are created at the organization level.

## Listing Resources

```bash
major resource list
```

Lists all resources in the organization as JSON. Each resource includes `isAttached` to show if it's connected to the current app. Example output:

```json
[{"id":"uuid","name":"My DB","type":"postgresql","description":"Production database","isAttached":true}]
```

## Adding a Resource to an App

```bash
major resource add --id "resource-uuid"
```

Adds a resource to the current application by ID. This:
1. Updates the app's resource list on the server
2. Generates the local client code via `major-client`
3. Installs dependencies

Use `major resource list` to find the resource ID.

## Removing a Resource from an App

```bash
major resource remove --id "resource-uuid"
```

Removes a resource from the current application. This:
1. Updates the app's resource list on the server
2. Removes the local client code

## Managing Resources (Interactive)

```bash
major resource manage
```

This is **interactive** -- the user must run it in their terminal. It opens a TUI menu to:
- View connected resources
- Connect new resources to the app
- Disconnect resources

## Environment Switching

### List Environments

```bash
major resource env-list
major resource env-list --json
```

Lists available environments. With `--json`, outputs:
```json
[{"id":"uuid","name":"production","isCurrent":true}]
```

### Switch Environment

```bash
major resource env --id "environment-uuid"
```

Switches to a specific environment non-interactively.

```bash
major resource env
```

Without `--id`, opens an interactive TUI picker.

Resources can be configured differently per environment (dev, staging, production).

## How Resources Work in Code

When you run `major app start`, environment variables are automatically injected for all connected resources. Your code accesses them via `process.env`:

```javascript
// Example: PostgreSQL connection
const dbUrl = process.env.DATABASE_URL;
```

The `.env` file is generated and refreshed automatically by `major app start` and `major app clone`.

## Adding Resources During App Creation

When creating an app with `major app create`, the CLI prompts to select resources. This connects them immediately and includes their environment variables in the generated `.env` file.

## Adding Resources After Creation

1. List available resources: `major resource list`
2. Add by ID: `major resource add --id "resource-uuid"`
3. Restart the dev server with `major app start` to pick up new environment variables
