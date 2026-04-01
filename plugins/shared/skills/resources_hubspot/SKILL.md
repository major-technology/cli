---
name: using-hubspot-connector
description: Implements HubSpot CRM data access for contacts, companies, and deals using generated clients and MCP tools. Use when doing ANYTHING that touches HubSpot in any way, load this skill.
---

# Major Platform Resource: HubSpot

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

- `mcp__resources__hubspot_get` — Make a GET request to any HubSpot API endpoint. Args: `resourceId`, `path`, `queryParams?`
- `mcp__resources__hubspot_list_contacts` — List contacts with optional property filtering. Args: `resourceId`, `properties?`, `limit?`
- `mcp__resources__hubspot_list_companies` — List companies with optional property filtering. Args: `resourceId`, `properties?`, `limit?`
- `mcp__resources__hubspot_list_deals` — List deals with optional property filtering. Args: `resourceId`, `properties?`, `limit?`

## TypeScript Client

```typescript
import { hubspotClient } from "./clients";

// invoke(method, path, invocationKey, options?)
const result = await hubspotClient.invoke("GET", "/crm/v3/objects/contacts", "fetch-contacts", {
	query: { limit: "10", properties: "firstname,lastname,email" },
});
if (result.ok && result.result.status === 200 && result.result.body.kind === "json") {
	const contacts = result.result.body.value.results;
}
```

## CRM Search API — [SEARCH.md](./SEARCH.md)

Read **SEARCH.md** before doing any CRM search. It covers:

- Filter operators (syntax, `value` vs `values` array, common mistakes)
- FilterGroup AND/OR logic and hard limits (5 groups, 6 filters/group, 18 total)
- **Date filters: epoch milliseconds as strings, NOT date strings**
- Case sensitivity rules (enums, `IN`/`NOT_IN` lowercase requirement)
- Sorting (1 rule max), pagination (200/page max, 10K total max)
- Association searching, searchable properties per object, rate limits (5 req/s)

---

## Tips

- **Use batch API calls when possible** — reduces API calls and avoids rate limits
- **Rate limits**: General API = 100 requests per 10 seconds. **Search API = 5 requests per second** (stricter). Respect `Retry-After` headers.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`
- Specify `properties` parameter to fetch only needed fields — improves performance

**Docs**: [HubSpot API Reference](https://developers.hubspot.com/docs/api/overview)
