---
name: using-zendesk-connector
description: Implements Zendesk Support API access (tickets, search, users, comments) through the Major HTTP proxy. Use when doing ANYTHING that touches Zendesk in any way, load this skill.
---

# Major Platform Resource: Zendesk

Zendesk is a proxy-only resource — every call goes through Major's HTTP proxy. There is **no typed `zendesk_*` MCP tool and no generated TypeScript client**. The proxy resolves the tenant host (`<subdomain>.zendesk.com`) at request time from the connected account, so you never hardcode or pass the subdomain — always pass a **leading-slash relative path** like `/api/v2/tickets.json`.

**Security**: Never set the `Authorization` header — the proxy injects it. Reserved headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`, `X-Major-*`) are stripped on the way out.

---

## MCP Tools

Use the generic HTTP proxy tools — there are no Zendesk-specific MCP tools.

- `mcp__resources__http_proxy_get` — Read-only GET. Args: `resourceId`, `url` (leading-slash path), `headers?`, `timeoutMs?`
- `mcp__resources__http_proxy_invoke` — Any HTTP method. Args: `resourceId`, `method`, `url` (leading-slash path), `headers?`, `body?`, `timeoutMs?`

See [using-http-proxy](../http-proxy/SKILL.md) for the full proxy reference.

## HTTP Proxy via `createProxyFetch` (Next.js)

For app code, drop the generic `createProxyFetch` into any SDK that takes a custom `fetch`, or use it as a plain `fetch` wrapper. Pass **leading-slash paths** instead of a full URL — the proxy resolves the upstream host.

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.ZENDESK_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

// List the 25 most recently updated tickets
const res = await proxyFetch("/api/v2/tickets.json?per_page=25&sort_by=updated_at&sort_order=desc");
if (!res.ok) throw new Error(`Zendesk ${res.status}: ${await res.text()}`);
const { tickets } = await res.json();

// Search across tickets / users / orgs / articles
const search = await proxyFetch(
  `/api/v2/search.json?query=${encodeURIComponent("type:ticket status<solved priority:high")}`,
);

// Create a ticket
const created = await proxyFetch("/api/v2/tickets.json", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    ticket: {
      subject: "Follow-up on incident #1234",
      comment: { body: "We're seeing this again — please advise." },
      priority: "high",
      tags: ["incident", "followup"],
    },
  }),
});

// Update a ticket
await proxyFetch("/api/v2/tickets/123.json", {
  method: "PUT",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ ticket: { status: "solved" } }),
});
```

**Rules:**

- `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they're set, don't ask the user to provide them. `ZENDESK_RESOURCE_ID` is the UUID of the connected Zendesk resource.
- **Always pass a leading-slash path.** Hardcoding `https://<sub>.zendesk.com/...` will fail — the subdomain isn't known to app code.
- **`resourceId` must be a static string literal** — the query extractor only tracks proxy calls when the resource id is known at build time.
- Server-side only (Server Components, Route Handlers, Server Actions). The `/next` subpath uses `next/headers`.
- Do NOT set the `Authorization` header — the proxy injects upstream auth.

## Common Zendesk endpoints

| What               | Method | Path                                                          |
| ------------------ | ------ | ------------------------------------------------------------- |
| List tickets       | GET    | `/api/v2/tickets.json`                                        |
| Get one ticket     | GET    | `/api/v2/tickets/{id}.json`                                   |
| Create a ticket    | POST   | `/api/v2/tickets.json` with body `{ "ticket": { ... } }`      |
| Update a ticket    | PUT    | `/api/v2/tickets/{id}.json` with body `{ "ticket": { ... } }` |
| Search             | GET    | `/api/v2/search.json?query=type:ticket+status<solved`         |
| Authenticated user | GET    | `/api/v2/users/me.json`                                       |
| List users         | GET    | `/api/v2/users.json`                                          |
| Ticket comments    | GET    | `/api/v2/tickets/{id}/comments.json`                          |

## Tips

- **Status filtering uses Zendesk search syntax**, not query params on `/tickets.json`. To filter by status, hit `/api/v2/search.json` with a `query` like `type:ticket status:open` or `status<solved`.
- **Write payloads are wrapped.** Zendesk expects bodies like `{ "ticket": { ... } }`, `{ "comment": { ... } }`, etc.
- **Pagination**: page-based on most endpoints (`page` + `per_page`). Cursor-based is available on tickets via `page[size]` / `page[after]`.
- **Default page size** is 100; pass `per_page` to reduce.
- **Non-2xx upstream statuses are passed through** — always inspect `res.ok` / `res.status` before treating the body as success.
- **Rate limits**: Zendesk's standard plan is 700 req/min per account. Prefer `/api/v2/search.json` for filtered reads over list-then-fetch loops.

**Docs**: [Zendesk REST API Reference](https://developer.zendesk.com/api-reference/ticketing/introduction/)
