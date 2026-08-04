---
name: using-managed-database
description: Understand and use Major-managed PostgreSQL databases. Use when the user wants a database, needs to store data, mentions "managed database", or asks about database setup.
---

# Major Platform: Managed Databases

## What is a Managed Database?

A managed database is a Major-hosted PostgreSQL instance provisioned and managed entirely by the platform. No credentials to manage, no connection strings to configure — everything is handled automatically. Once active, it appears as a regular PostgreSQL resource with `isManaged: true`.

## App-Scoped vs Org-Scoped

There are two types of managed databases:

- **App-scoped**: Belongs to a single application. Permissions are automatically inherited from app roles — app admins and editors get `Resource:Admin` on the database. Other apps cannot access it.
- **Org-scoped**: Shared across all apps in the organization. Created by org admins through the dashboard. Visible to all apps with appropriate permissions.

## Lifecycle and Migrations

Managed database lifecycle and migration tools have moved to orchestrator/build surfaces; they are not resource MCP tools. Do not invent or call a resource MCP setup, provisioning, deprovisioning, or migration tool. Use the lifecycle tools exposed by the current orchestrator/build context. Once a managed PostgreSQL resource is connected, use the resource MCP tools below for SQL.

## Using the Database Once Active

After the database is connected and you have its resource ID:

1. **MCP tools** (direct SQL, no code needed):
   - `mcp__resources__postgresql_psql` — Read-only SQL queries and psql commands (`\dt`, `\d`, etc.). Args: `resourceId`, `command`, `timeoutMs?`
   - `mcp__resources__postgresql_invoke` — DDL/DML and other write queries. Args: `resourceId`, `sql`, `params?`, `timeoutMs?`. Use `$1`, `$2`, ... placeholders in `sql` and pass their values positionally in `params`.

2. **Generated TypeScript clients** (for app code):
   - Call `mcp__resource-tools__add-resource-client` with the `resourceId` to generate a typed PostgreSQL client
   - Use the client for read/write operations in your application code

## Identifying Managed Databases

In `mcp__resources__list_resources`, managed databases have `isManaged: true` and a `managedScope` field:

- `managedScope: "app"` — App-scoped, belongs to this application only
- `managedScope: "org"` — Org-scoped, shared across all apps in the organization

Regular (external) PostgreSQL resources have `isManaged: false`.

## Choosing Between App and Org Databases

If the user has both an app-scoped and an org-scoped managed database available, **ask the user which one they want to use** before proceeding. Do not assume. For example: "I see you have both an app database and an organization-wide database. Which one should I use for this task?"

If there is only an app-scoped database, use it without asking. If there is only an org-scoped database, ask whether the user wants to use it or provision an app-scoped database through the available orchestrator/build workflow. Generally, prefer an app database unless the data should be shared across the organization.

## Tips

- Use `postgresql_psql` for read-only exploration (schema inspection, SELECT queries)
- Use `postgresql_invoke` for schema changes and data modifications on connected managed databases
- Use parameterized queries (`$1`, `$2`, ...) — never interpolate values into SQL strings
- After creating tables, generate a TypeScript client for the app to use in code
