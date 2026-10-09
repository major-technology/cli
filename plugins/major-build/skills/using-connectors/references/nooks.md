# Major Platform Resource: Nooks API

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Three ways to interact with Nooks:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__plugin_major_major__sandbox_add-resource-client` with the app's `slug` and a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).
3. **HTTP proxy** (Next.js apps): Use `createProxyFetch` from `@major-tech/resource-client/next` to call the Nooks API directly with automatic auth injection. See [using-http-proxy](http-proxy.md) for setup and usage. Relative paths are joined to `https://partner-api.nooks.in/v1`, so pass `/users`, not `/v1/users`. For `_href` or `links.next` values that start with `/v1`, prefix `https://partner-api.nooks.in` to make them absolute.

**CRITICAL: Do NOT guess client method names or signatures.** ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-nooks-users"`, never dynamic values.

---

## MCP Tools

- `mcp__resources__nooks_get` — Make a GET request to any Nooks API endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__nooks_invoke` — Make any HTTP request to the Nooks API (including writes). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { nooksClient } from "./clients";

// invoke(method, path, invocationKey, options?)
const result = await nooksClient.invoke("GET", "/users", "list-nooks-users", {
    query: { "page[size]": "100", "filter[email]": "jane@example.com" },
});
if (result.ok) {
    const { data, links } = result.result.body.value as { data: unknown[]; links: { next: string | null } };
}

// Enroll a prospect in a sequence (owner.id is a Nooks user ID from GET /users)
await nooksClient.invoke("POST", "/sequenceStates", "enroll-prospect", {
    body: {
        type: "json",
        value: { data: { prospect: { id: prospectId }, sequence: { id: sequenceId }, owner: { id: userId } } },
    },
});
```

## Tips

- **Base URL is `https://partner-api.nooks.in/v1`**. Write paths the way the [Nooks API reference](https://developer.nooks.in) shows them (`/users`, `/sequences`, `/calls`). `_href` values from responses (`/v1/...`) and full `links.next` URLs are also accepted as the path.
- **Auth is automatic.** The workspace API key (`nooks-api-...`) is sent as `Authorization: Bearer`. Never set it yourself. API keys have full read/write access to the workspace (no scopes).
- **Check the connection** with `GET /me`. It returns the workspace and the user who created the key.
- **Two APIs share the same base URL:**
  - **Sequencing API:** `/sequences`, `/sequenceSteps`, `/sequenceStates` (enrollments), `/prospects`, `/accounts`, `/users`, `/tasks`, `/calls` (dialer calls), `/callDispositions`, `/emails`, `/mailboxes`, `/tags`.
  - **Conversations API (beta):** `/conversations`, `/transcripts`, `/scorecards`, `/teams`, `/opportunities`. It is enabled per workspace on request, and most endpoints need a Conversation Intelligence seat. Conversations (meeting recordings) are a separate data set from dialer `/calls`.
- **Pagination** is cursor-based. Use `page[size]` (default 50, max 100). Follow `links.next`, and stop when it is `null`. Never build `page[after]` cursors yourself. In the Sequencing API `links.next` is a full URL; in the Conversations API it is a `/v1/...` path.
- **Filters** use bracket query params, e.g. `filter[email]`, `filter[updatedAt][gte]`. Check each endpoint in the reference for the filters it supports.
- **`include`** expands related references inline (GET only, max 3 comma-separated fields, no nesting), e.g. `GET /prospects/{id}?include=sequenceStates`.
- **Rate limits** are per workspace and endpoint, per minute. Typical limits are 300 for list reads, 600 for reads by ID, and 120 for most writes; some endpoints are lower (e.g. `/transcripts` is 60, CRM notes are 30). A `429` comes with `Retry-After`.
- **Errors** come back as `{ "error": { "code", "message" }, "traceId" }`. An invalid key returns `401` with code `INVALID_API_KEY`.
