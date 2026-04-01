---
name: using-outreach-connector
description: Implements Outreach prospect and sequence management using generated clients and MCP tools. Use when doing ANYTHING that touches Outreach or Outreach.io in any way, load this skill.
---

# Major Platform Resource: Outreach

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

- `mcp__resources__outreach_get` — Make a GET request to any Outreach API endpoint. Args: `resourceId`, `path`, `queryParams?`
- `mcp__resources__outreach_list_prospects` — List prospects with pagination. Args: `resourceId`, `limit?`
- `mcp__resources__outreach_list_sequences` — List sequences with pagination. Args: `resourceId`, `limit?`

## TypeScript Client

```typescript
import { outreachClient } from "./clients";

// invoke(method, path, invocationKey, options?)
const result = await outreachClient.invoke("GET", "/api/v2/prospects", "list-prospects", {
	queryParams: { "page[limit]": "10" },
});
if (result.ok) {
	console.log(result.result.data);
}

// Create a prospect — uses JSON:API format
await outreachClient.invoke("POST", "/api/v2/prospects", "create-prospect", {
	body: {
		data: {
			type: "prospect",
			attributes: { firstName: "John", lastName: "Doe", emails: ["john@example.com"] },
		},
	},
});
```

## Tips

- **Request bodies use JSON:API format**: `{ data: { type: "...", attributes: {...} } }`
- Pagination uses `page[limit]` and `page[offset]` query parameters
- Paths include the full API path: `/api/v2/prospects`, `/api/v2/sequences`, etc.

**Docs**: [Outreach API Reference](https://developers.outreach.io/api/reference/)
