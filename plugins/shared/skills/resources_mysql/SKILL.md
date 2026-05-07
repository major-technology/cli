---
name: using-mysql-connector
description: Implements MySQL database connections, SQL queries, and data operations using generated clients and MCP tools. Use when doing ANYTHING that touches MySQL in any way, load this skill.
---

# Major Platform Resource: MySQL

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

- `mcp__resources__mysql_query` — Execute a read-only MySQL query. Supports SELECT and introspection statements like `SHOW TABLES`, `DESCRIBE table_name`, `SHOW DATABASES`, `SHOW CREATE TABLE table_name`, and `EXPLAIN`. **Does not require user approval — prefer this tool for all read-only operations.** Args: `resourceId`, `statement`, `params?`, `description`, `timeoutMs?`
- `mcp__resources__mysql_invoke` — Execute any SQL statement including write operations (INSERT, UPDATE, DELETE, CREATE TABLE, ALTER TABLE, DROP, etc.). Returns rows and rowsAffected. **Requires user approval — only use when you need to write data.** Args: `resourceId`, `sql`, `params?`, `description`, `timeoutMs?`

**IMPORTANT: Always use `mysql_query` for read-only operations.** It does not require user approval, making the workflow faster and smoother. Only use `mysql_invoke` when you actually need to perform writes (INSERT, UPDATE, DELETE, DDL). Never use `mysql_invoke` for SELECT queries or schema exploration.

## TypeScript Client

```typescript
import { myMysqlClient } from "./clients";

// invoke<T>(sql, params?, invocationKey, timeoutMs?)
// Uses positional ? placeholders
const result = await myMysqlClient.invoke<{ id: number; name: string }>(
	"SELECT * FROM users WHERE id = ?",
	[userId],
	"fetch-user",
);
if (result.ok) {
	console.log(result.result.rows);
}
```

## Tips

- **Use `mysql_query` exclusively for read-only tasks. Never use `mysql_invoke` for read-only.**
- Uses **positional `?` placeholders** — not `$1, $2` like PostgreSQL or `@name` like MSSQL. The first `?` maps to `params[0]`, the second to `params[1]`, etc.
- Default port is **3306**
- Use `mysql_query` with `SHOW DATABASES`, `SHOW TABLES`, `DESCRIBE table_name`, `SHOW CREATE TABLE table_name`, `SHOW INDEX FROM table_name` to explore database structure
- JSON columns (MySQL 5.7+) are returned as parsed objects, not raw strings
- Use backticks (`` ` ``) for identifiers (table/column names), single quotes for string values
- `LIMIT n` for pagination (not `FETCH FIRST n ROWS ONLY`). For offset: `LIMIT offset, count` or `LIMIT count OFFSET offset`
- Table names are case-sensitive on Linux, case-insensitive on macOS/Windows — always use exact case from the schema

**Docs**: [MySQL 8.0 Reference Manual](https://dev.mysql.com/doc/refman/8.0/en/)
