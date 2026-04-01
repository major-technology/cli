---
name: using-custom-api-connector
description: Implements custom REST API HTTP requests with automatic auth header injection using generated clients and MCP tools. Use when doing ANYTHING that touches a custom API resource in any way, load this skill.
---

# Major Platform Resource: Custom REST API

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

- `mcp__resources__custom_get` — Make a GET request to the configured API endpoint. Args: `resourceId`, `path`, `queryParams?`

## TypeScript Client

```typescript
import { apiClient } from "./clients";

// invoke(method, path, invocationKey, options?)
const result = await apiClient.invoke("GET", "/users", "list-users", { query: { page: "1", limit: "20" } });
if (result.ok) {
	const response = result.result;
	// response: { kind: "api", status: number, body: { kind: "json"|"text"|"binary", value: ... } }
}

// POST with body
await apiClient.invoke("POST", "/users", "create-user", {
	body: { type: "json", value: { name: "Jane", email: "jane@example.com" } },
});
```

## Tips

- **Paths are relative to the resource's configured base URL**
- **Auth headers are automatically injected** — the resource configuration includes secret headers (e.g., Authorization) that you don't need to set manually
- Supports all HTTP methods: GET, POST, PUT, PATCH, DELETE
- Custom headers can be added via the `headers` option in the TypeScript client

**Docs**: Refer to the specific API's documentation for endpoint details.
