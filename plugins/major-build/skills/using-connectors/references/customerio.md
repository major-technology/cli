# Major Platform Resource: Customer.io API

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Three ways to interact with Customer.io:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__plugin_major_major__sandbox_add-resource-client` with the app's `slug` and a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).
3. **HTTP proxy** (Next.js apps): Use `createProxyFetch` from `@major-tech/resource-client/next` to call Customer.io directly with automatic auth injection. See [using-http-proxy](http-proxy.md) for setup and usage. Relative paths are joined to the App API host, so pass `/v1/segments`. For the Track API, pass the full URL (`https://track.customer.io/api/v1/...`, or `https://track-eu.customer.io/...` for EU accounts).

**CRITICAL: Do NOT guess client method names or signatures.** ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-customerio-segments"`, never dynamic values.

---

## MCP Tools

- `mcp__resources__customerio_get` — Make a GET request to the App API. Args: `resourceId`, `path`, `query?`
- `mcp__resources__customerio_invoke` — Make any HTTP request to the App API or Track API (including writes). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { customerioClient } from "./clients";

// invoke(method, path, invocationKey, options?)
// App API: look up people by email
const result = await customerioClient.invoke("GET", "/v1/customers", "find-customerio-person", {
    query: { email: ["jane@example.com"] },
});
if (result.ok) {
    const { results } = result.result.body.value as { results: { cio_id: string; email: string; id: string }[] };
}

// Track API: add or update a person, then track an event
await customerioClient.invoke("PUT", `/api/v1/customers/${userId}`, "identify-customerio-person", {
    body: { type: "json", value: { email: "jane@example.com", plan: "basic" } },
});
await customerioClient.invoke("POST", `/api/v1/customers/${userId}/events`, "track-customerio-event", {
    body: { type: "json", value: { name: "purchase", data: { price: 23.45, product: "socks" } } },
});

// App API: send a transactional email from a template
await customerioClient.invoke("POST", "/v1/send/email", "send-customerio-receipt", {
    body: {
        type: "json",
        value: {
            transactional_message_id: 44,
            to: "jane@example.com",
            identifiers: { email: "jane@example.com" },
            message_data: { order_id: "123" },
        },
    },
});
```

## Tips

- **Write paths exactly as the [Customer.io API reference](https://docs.customer.io/integrations/api/) shows them.** The path picks the API:
  - **App API** (`/v1/...`, Bearer App API key): reads people, segments, campaigns, broadcasts, messages, and exports; sends transactional messages (`POST /v1/send/email`, `/v1/send/push`, `/v1/send/sms`, ...) and triggers broadcasts (`POST /v1/campaigns/{broadcast_id}/triggers`).
  - **Track API** (`/api/v1/...`, `/api/v2/...`, Basic site ID + API key): adds or updates people (`PUT /api/v1/customers/{identifier}`), tracks events (`POST /api/v1/customers/{identifier}/events`), and sends batched v2 calls (`POST /api/v2/entity`, `POST /api/v2/batch`).
- **Auth and region are automatic.** The connector stores the account's region (US or EU) and sends each request to the right host with the right credentials. Never set `Authorization` yourself. A connector may have only one of the two credential sets; a call to the other API fails with a "not configured" error.
- **Credentials belong to one workspace.** To reach another workspace, connect another resource.
- **Check the connection** with `GET /v1/workspaces` (App API). On the Track API, `GET /api/v1/accounts/region` returns the account's region.
- **Find people** with `GET /v1/customers?email=...`, or `POST /v1/customers` with a `filter` body (`and` / `or` / `not` of `segment` and `attribute` conditions) for up to 1000 people per request. Read one person with `GET /v1/customers/{customer_id}/attributes`. Pass `id_type=email` (or `cio_id`, `phone`) when `customer_id` is not the person's `id`.
- **Pagination**: list endpoints take `limit` (default 50, max 1000) and `start`. Pass the response's `next` value as `start` to get the next page.
- **Track API identifiers**: `{identifier}` can be an `id`, an email address, or a `cio_id` prefixed with `cio_`. To identify people by phone number, use the Track v2 API.
- **Rate limits**: most App API endpoints allow 10 requests per second and return `429` with `Retry-After`. `POST /v1/campaigns/{broadcast_id}/triggers` allows 1 request every 10 seconds. The Track API allows 1000 requests per second (not strictly enforced).
