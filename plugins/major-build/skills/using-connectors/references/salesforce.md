# Major Platform Resource: Salesforce

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Three ways to interact with Salesforce:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__plugin_major_major__sandbox_add-resource-client` with the app's `slug` and a `resourceId` to generate a typed client. Clients are created in `backend/src/clients/` (`/clients/` in an older Next.js app).
3. **HTTP proxy** (app backend code): Use `createProxyFetch` to call the Salesforce API directly with automatic auth injection. See [using-http-proxy](http-proxy.md) for setup and usage — preferred when you need to hit endpoints not covered by MCP tools or the typed client, or when using an official SDK that accepts a custom `fetch`.

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated clients directory (`backend/src/clients/`) or the package itself to verify available methods and their exact signatures before writing any client code.

**Framework note**: use resource clients in backend code only (`backend/src/`), never in the frontend; the frontend calls your own `/api` route. In an older Next.js app: server-side code only (Server Components, Server Actions, Route Handlers).

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__salesforce_get` — Make a GET request to any Salesforce API endpoint. Args: `resourceId`, `path`, `queryParams?`
- `mcp__resources__salesforce_query` — Execute a SOQL query. Args: `resourceId`, `query`
- `mcp__resources__salesforce_describe_object` — Get metadata and field definitions for an sObject. Args: `resourceId`, `objectType`

## TypeScript Client

```typescript
import { sfClient } from "./clients";

// Prefer helper methods over raw invoke()
const result = await sfClient.query(
	"SELECT Id, Name FROM Account WHERE CreatedDate > 2024-01-01T00:00:00Z LIMIT 10",
	"recent-accounts",
);

// CRUD helpers
await sfClient.getRecord("Account", recordId, "get-account", { fields: ["Name", "Industry"] });
await sfClient.createRecord("Account", { Name: "Acme Corp" }, "create-account");
await sfClient.updateRecord("Account", recordId, { Name: "Updated" }, "update-account");
await sfClient.deleteRecord("Account", recordId, "delete-account");
await sfClient.describeObject("Account", "describe-account");
```

## Tips

- **Use `query()` helper for SOQL** — cleaner than building the path manually
- **Governor limits**: Be mindful of API call limits (varies by org edition). Use bulk API for large data operations.
- Use `describeObject()` to explore field names and types before writing queries
- Salesforce API paths include the version: `/services/data/v63.0/...`

**Docs**: [Salesforce REST API Reference](https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/)
