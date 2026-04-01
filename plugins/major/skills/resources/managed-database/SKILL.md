---
name: using-managed-database
description: Set up and use Major-managed PostgreSQL databases. Use when the user wants a database, needs to store data, mentions "managed database", or asks about database setup.
---

# Major Platform: Managed Databases

## What is a Managed Database?

A managed database is a Major-hosted PostgreSQL instance provisioned and managed entirely by the platform. No credentials to manage, no connection strings to configure — everything is handled automatically. Once active, it appears as a regular PostgreSQL resource with `isManaged: true`.

## App-Scoped vs Org-Scoped

There are two types of managed databases:

- **App-scoped** (created by this tool): Belongs to a single application. Permissions are automatically inherited from app roles — app admins and editors get `Resource:Admin` on the database. Other apps cannot access it.
- **Org-scoped**: Shared across all apps in the organization. Created by org admins through the dashboard. Visible to all apps with appropriate permissions.

The `setup_managed_database` MCP tool creates **app-scoped** databases only.

## Setting Up a Managed Database

Call `mcp__resources__setup_managed_database` — no arguments needed. The tool automatically provisions a database for the current application.

**Behavior:**

- **First call** (no database exists): Starts provisioning. Takes around 1 minute.
- **While provisioning**: Returns status. Wait ~1 minute and call again.
- **Once active**: Returns the resource ID. The database is ready to use.
- **If failed**: Returns failure status. Deprovision and try again.

## Using the Database Once Active

After setup completes and you have the resource ID:

1. **MCP tools** (direct SQL, no code needed):
   - `mcp__resources__postgresql_psql` — Read-only SQL queries and psql commands (`\dt`, `\d`, etc.). Args: `resourceId`, `command`
   - `mcp__resources__postgresql_run_migration` — DDL/DML migrations (managed databases only). Args: `resourceId`, `migration`, `description?`

2. **Generated TypeScript clients** (for app code):
   - Call `mcp__resource-tools__add-resource-client` with the `resourceId` to generate a typed PostgreSQL client
   - Use the client for read/write operations in your application code

## Identifying Managed Databases

In `mcp__resources__list_resources`, managed databases have `isManaged: true` and a `managedScope` field:

- `managedScope: "app"` — App-scoped, belongs to this application only
- `managedScope: "org"` — Org-scoped, shared across all apps in the organization

The `postgresql_run_migration` tool only works on managed databases. Regular (external) PostgreSQL resources have `isManaged: false`.

## Choosing Between App and Org Databases

If the user has both an app-scoped and an org-scoped managed database available, **ask the user which one they want to use** before proceeding. Do not assume. For example: "I see you have both an app database and an organization-wide database. Which one should I use for this task?"

If there is only an app-scoped database just use that one, don't ask. If there is only an org-scoped database, ask the user if they'd like to make a new app-scoped db. Generally, it's better to use an app DB unless there's a real
reason that data that should be shared for the entire org.

## Tips

- Use `postgresql_psql` for read-only exploration (schema inspection, SELECT queries)
- Use `postgresql_run_migration` for all schema changes and data modifications on managed databases
- Use parameterized queries (`$1`, `$2`, ...) — never interpolate values into SQL strings
- After creating tables with `run_migration`, generate a TypeScript client for the app to use in code
