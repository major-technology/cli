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

## Managing Resources

```bash
major resource manage
```

This is **interactive** — the user must run it in their terminal. It opens a TUI menu to:
- View connected resources
- Connect new resources to the app
- Disconnect resources

## Environment Switching

```bash
major resource env
```

View or switch between available environments for the app. Resources can be configured differently per environment (dev, staging, production).

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

1. Run `major resource manage` (interactive — user must run in terminal)
2. Select resources to connect
3. Restart the dev server with `major app start` to pick up new environment variables
