---
name: using-postgresql-connector
description: Implements PostgreSQL connections, SQL queries, and migration patterns using generated clients and MCP tools. Use when doing ANYTHING that touches PostgreSQL, Postgres, pg, or psql in any way, load this skill.
---

# Major Platform Resource: PostgreSQL

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "List all user accounts", "Check table schema"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__postgresql_psql` — Execute read-only SQL queries and psql backslash commands (`\dt`, `\d`, `\di`, `\df`, etc.). Args: `resourceId`, `command`, `timeoutMs?`
- `mcp__resources__postgresql_run_migration` — Run DDL/DML migrations on **managed databases only** (`isManaged=true`). Runs in a transaction; rolls back on failure. Args: `resourceId`, `migration`, `description?`

## TypeScript Client

```typescript
import { myDbClient } from "./clients";

// invoke<T>(sql, params?, invocationKey, timeoutMs?)
const result = await myDbClient.invoke<{ id: number; name: string }>(
	"SELECT * FROM users WHERE id = $1",
	[userId],
	"fetch-user",
);
if (result.ok) {
	console.log(result.result.rows);
}
```

## Tips

- Use parameterized queries (`$1`, `$2`, ...) — never interpolate values into SQL strings
- `psql` tool is read-only; use `run_migration` for writes (managed DBs) or the TypeScript client for writes (external DBs)
- The TypeScript client supports full read/write operations regardless of managed status
- Use `psql` exclusively for read-only tasks. Never use invoke for read only.

**Docs**: [PostgreSQL Documentation](https://www.postgresql.org/docs/)
