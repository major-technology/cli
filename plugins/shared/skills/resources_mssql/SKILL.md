---
name: using-mssql-connector
description: Implements Microsoft SQL Server connections, queries, and schema exploration using generated clients and MCP tools. Use when doing ANYTHING that touches MSSQL, SQL Server, or T-SQL in any way, load this skill.
---

# Major Platform Resource: Microsoft SQL Server

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

- `mcp__resources__mssql_list_schemas` — List all schemas (excludes system schemas). Args: `resourceId`
- `mcp__resources__mssql_list_tables` — List tables, optionally filtered by schema. Args: `resourceId`, `schema?`
- `mcp__resources__mssql_list_columns` — List columns with types/constraints for a table. Args: `resourceId`, `schema`, `table`
- `mcp__resources__mssql_query` — Execute read-only SQL (SELECT/WITH only). Args: `resourceId`, `statement`, `params?`

## TypeScript Client

```typescript
import { myMssqlClient } from "./clients";

// invoke<T>(sql, params?, invocationKey, timeoutMs?)
// Uses named parameters: @paramName
const result = await myMssqlClient.invoke<{ id: number; name: string }>(
	"SELECT * FROM users WHERE id = @id",
	{ id: userId },
	"fetch-user",
);
if (result.ok) {
	console.log(result.result.rows);
}
```

## Tips

- Uses **named parameters** (`@id`, `@name`) not positional (`$1`)
- MCP query tool only allows SELECT/WITH — no multi-statement queries
- Use `list_schemas` → `list_tables` → `list_columns` to explore database structure

**Docs**: [SQL Server Documentation](https://learn.microsoft.com/en-us/sql/sql-server/)
