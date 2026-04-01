---
name: using-snowflake-connector
description: Implements Snowflake warehouse queries, schema exploration, and data operations using generated clients and MCP tools. Use when doing ANYTHING that touches Snowflake in any way, load this skill.
---

# Major Platform Resource: Snowflake

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__snowflake_list_databases` — List all accessible databases. Args: `resourceId`
- `mcp__resources__snowflake_list_schemas` — List schemas in a database. Args: `resourceId`, `database`
- `mcp__resources__snowflake_list_tables` — List tables in a schema. Args: `resourceId`, `database`, `schema`
- `mcp__resources__snowflake_query` — Execute read-only SQL (SELECT/SHOW/DESCRIBE/EXPLAIN). Args: `resourceId`, `statement`, `database?`, `schema?`

## TypeScript Client

```typescript
import { snowflakeClient } from "./clients";

// execute(statement, invocationKey, options?)
const result = await snowflakeClient.execute(
	"SELECT * FROM orders WHERE order_date > '2024-01-01' LIMIT 100",
	"recent-orders",
	{ database: "ANALYTICS", schema: "PUBLIC" },
);

// status(statementHandle, invocationKey, options?) — for async queries
// cancel(statementHandle, invocationKey)
```

## Tips

- MCP tools are **all read-only** — use the TypeScript client for write operations
- Use `list_databases` → `list_schemas` → `list_tables` to explore data warehouse structure
- The `query` tool accepts optional `database` and `schema` context parameters
- For long-running queries via the TypeScript client, use async execution with `status()` polling

**Docs**: [Snowflake Documentation](https://docs.snowflake.com/)
