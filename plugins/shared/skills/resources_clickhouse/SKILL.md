---
name: using-clickhouse-connector
description: Implements ClickHouse database connections, SQL queries, and data operations using generated clients and MCP tools. Use when doing ANYTHING that touches ClickHouse in any way, load this skill.
---

# Major Platform Resource: ClickHouse

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

- `mcp__resources__clickhouse_query` — Execute a read-only ClickHouse query. Supports SELECT and introspection statements like `SHOW DATABASES`, `SHOW TABLES`, `DESCRIBE TABLE table_name`, `SHOW CREATE TABLE table_name`, `EXISTS TABLE table_name`, and `EXPLAIN`. **Does not require user approval — prefer this tool for all read-only operations.** Args: `resourceId`, `statement`, `params?`, `description`, `timeoutMs?`
- `mcp__resources__clickhouse_invoke` — Execute any SQL statement including write operations (INSERT, CREATE TABLE, ALTER TABLE, DROP, etc.). Returns rows and rowsAffected. **Requires user approval — only use when you need to write data.** Args: `resourceId`, `sql`, `params?`, `description`, `timeoutMs?`

**IMPORTANT: Always use `clickhouse_query` for read-only operations.** It does not require user approval, making the workflow faster and smoother. Only use `clickhouse_invoke` when you actually need to perform writes (INSERT, DDL, mutations). Never use `clickhouse_invoke` for SELECT queries or schema exploration.

## TypeScript Client

```typescript
import { myClickhouseClient } from "./clients";

// invoke<T>(sql, params?, invocationKey, timeoutMs?)
// Uses positional ? placeholders
const result = await myClickhouseClient.invoke<{ id: number; name: string }>(
	"SELECT * FROM users WHERE id = ?",
	[userId],
	"fetch-user",
);
if (result.ok) {
	console.log(result.result.rows);
}
```

## Tips

- **Use `clickhouse_query` exclusively for read-only tasks. Never use `clickhouse_invoke` for read-only.**
- Uses **positional `?` placeholders** — not `$1, $2` like PostgreSQL or `@name` like MSSQL. The first `?` maps to `params[0]`, the second to `params[1]`, etc.
- Default HTTP port is **8123**, HTTPS port is **8443**
- Default username is `"default"`, default database is `"default"`
- Use `clickhouse_query` with `SHOW DATABASES`, `SHOW TABLES`, `DESCRIBE TABLE table_name`, `SHOW CREATE TABLE table_name`, `EXISTS TABLE table_name` to explore database structure
- ClickHouse is a **columnar analytics database** — avoid `SELECT *` on large tables, always use `LIMIT`
- ClickHouse does **NOT** support `UPDATE`/`DELETE` on regular MergeTree tables — use `ALTER TABLE ... UPDATE/DELETE` for mutations
- String comparison is **case-sensitive** by default; use `lower()` or `ilike` for case-insensitive matching
- `LIMIT n` for pagination (not `FETCH FIRST n ROWS ONLY`). For offset: `LIMIT n OFFSET m`

**Docs**: [ClickHouse SQL Reference](https://clickhouse.com/docs/en/sql-reference)
