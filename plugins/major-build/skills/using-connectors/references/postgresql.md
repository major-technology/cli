# Major Platform Resource: PostgreSQL

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__plugin_major_major__sandbox_add-resource-client` with the app's `slug` and a `resourceId` to generate a typed client. Clients are created in `backend/src/clients/` (`/clients/` in an older Next.js app).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated clients directory (`backend/src/clients/`) or the package itself to verify available methods and their exact signatures before writing any client code.

**Framework note**: use resource clients in backend code only (`backend/src/`), never in the frontend; the frontend calls your own `/api` route. In an older Next.js app: server-side code only (Server Components, Server Actions, Route Handlers).

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__postgresql_query` — Execute one read-only SQL statement (SELECT, SHOW, EXPLAIN). Inspect schema via `information_schema` / `pg_catalog`. Args: `resourceId`, `statement`, `params?`, `timeoutMs?`
- `mcp__resources__postgresql_invoke` — Execute write SQL against managed or external PostgreSQL resources. Args: `resourceId`, `sql`, `params?`, `timeoutMs?`, `description`

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
- `postgresql_query` is read-only; use `postgresql_invoke` for data modifications against managed or external PostgreSQL resources
- The TypeScript client supports full read/write operations regardless of managed status
- Use `postgresql_query` exclusively for read-only tasks. Never use invoke for read only.

**Docs**: [PostgreSQL Documentation](https://www.postgresql.org/docs/)
