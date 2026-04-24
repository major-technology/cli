---
name: using-attio-connector
description: Implements Attio CRM data access for people, companies, lists, notes, and tasks using generated clients and MCP tools. Use when doing ANYTHING that touches Attio in any way, load this skill.
---

# Major Platform Resource: Attio

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** After generating a client, ALWAYS read the generated client file to discover the exact API. Different resource types have completely different methods and calling conventions.

**Invocation keys** must be static string literals (e.g., `"list-objects"`). Never use dynamic values like template literals, variables, or `.toString()`.

---

## MCP Tools

- `mcp__resources__attio_get` — Make a GET request to any Attio API endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__attio_list_objects` — List all CRM objects in the workspace. Args: `resourceId`
- `mcp__resources__attio_list_records` — Query records for a specific object with optional filtering. Args: `resourceId`, `objectSlug`, `filter?`, `sorts?`, `limit?`
- `mcp__resources__attio_list_lists` — List all lists in the workspace. Args: `resourceId`
- `mcp__resources__attio_invoke` — Make any HTTP request to the Attio API (including writes). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

The Attio client uses a **flattened JSON response** (like Stripe). Check `.ok`, then access `.json` directly — no need to dig through `result.body.kind`.

```typescript
import { attioClient } from "./clients";

// invoke<T>(method, path, invocationKey, options?)
const result = await attioClient.invoke<{ data: Array<{ id: { object_id: string } }> }>(
    "GET", "/v2/objects", "list-objects"
);
if (result.ok) {
    const objects = result.json.data; // Typed as the generic T
}

// POST with body — create a person record
const createResult = await attioClient.invoke("POST", "/v2/objects/people/records", "create-person", {
    body: { type: "json", value: { data: { values: { name: [{ first_name: "Jane", last_name: "Doe" }] } } } },
});
if (createResult.ok) {
    console.log("Created:", createResult.json);
}

// Query records with filter
const queryResult = await attioClient.invoke("POST", "/v2/objects/companies/records/query", "query-companies", {
    body: { type: "json", value: { limit: 10 } },
});
```

---

## Tips

- **Attio API base URL**: `https://api.attio.com` — all paths start with `/v2/`
- **Standard objects**: `people`, `companies`, `deals`, `workspaces` — use slugs not IDs for standard objects
- **Record queries** use POST to `/v2/objects/{slug}/records/query` with filter/sort in the body
- **Pagination**: Use `offset` and `limit` params. Default limit is 25, max is 500.
- **Rate limits**: Attio uses a token bucket approach. Check `X-RateLimit-Remaining` header.
- Response structure with the flattened client: `{ ok: true, status: number, json: T }` — no nested body parsing needed.

**Docs**: [Attio REST API Reference](https://docs.attio.com/rest-api/overview)
