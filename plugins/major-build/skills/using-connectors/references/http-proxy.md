# Major Platform: HTTP Proxy

The HTTP proxy lets apps and MCP clients call any HTTP-based resource through a single endpoint — `<METHOD> /v1/proxy/<resourceId>` with the upstream URL in `X-Major-Target-URL`. The proxy validates the URL against the resource's policy, injects upstream auth, strips reserved headers, and streams the response back.

**Use this when:**

- You want to use a third-party SDK (Stripe, OpenAI, Twilio, etc.) that accepts a custom `fetch` and have it route through a connected resource.
- You need to hit an upstream HTTP endpoint that isn't exposed by the typed resource client or a specialized MCP tool.
- You're working with a resource that has no generated client (proxy-only resources).

**Don't use this when** a specialized MCP tool (e.g. `stripe_list_customers`, `hubspot_get`) or a typed resource client method covers what you need — those are preferred for typed inputs/outputs.

**Security:** Never set the `Authorization` header — the proxy injects upstream auth. The proxy strips reserved request headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`) and the `X-Major-*` / `X-Pd-*` namespaces.

---

## SDK: `createProxyFetch`

`createProxyFetch` returns a `fetch`-compatible function that routes every request through the Major proxy. Drop it into any SDK that accepts a custom `fetch`. Use it in backend code only (`backend/src/`).

**Import:**

```typescript
import { createProxyFetch } from "@major-tech/resource-client";
import { tryGetContext } from "hono/context-storage";
```

**Config:**

| Field           | Type           | Required | Notes                                                                                         |
| --------------- | -------------- | -------- | --------------------------------------------------------------------------------------------- |
| `baseUrl`       | `string`       | yes      | Major API base — use `process.env.MAJOR_API_BASE_URL`                                         |
| `resourceId`    | `string`       | yes      | UUID of the resource to proxy through                                                         |
| `majorJwtToken` | `string`       | yes      | App-level JWT — use `process.env.MAJOR_JWT_TOKEN`                                             |
| `getUserJwt`    | `() => string \| null \| undefined` | no | Returns the signed-in user's `x-major-user-jwt`; required for per-user-OAuth resources |
| `fetch`         | `typeof fetch` | no       | Override runtime fetch (defaults to `globalThis.fetch`)                                       |
| `timeoutMs`     | `number`       | no       | Default `X-Major-Timeout-Ms` (server-clamped to 60_000); only set if caller didn't supply one |

`MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they are already set; do not ask the user to provide them.

Pass `getUserJwt: () => tryGetContext()?.req.header("x-major-user-jwt")` so per-user-OAuth resources (Gmail, Calendar, Drive) resolve to the signed-in user. It reads the current request through `contextStorage()`, which the template registers in `backend/src/app.ts`. Outside a request (no context) it returns nothing and the header is skipped.

### Drop-in with a third-party SDK

```typescript
import Stripe from "stripe";
import { createProxyFetch } from "@major-tech/resource-client";
import { tryGetContext } from "hono/context-storage";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.STRIPE_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
  getUserJwt: () => tryGetContext()?.req.header("x-major-user-jwt"),
});

const stripe = new Stripe("sk_unused_proxy_injects_real_key", {
  httpClient: Stripe.createFetchHttpClient(proxyFetch),
});

const customers = await stripe.customers.list({ limit: 10 });
```

The placeholder key is required by the Stripe SDK constructor but is never sent — the proxy strips `Authorization` and injects the real key from the connected resource.

### Plain fetch

```typescript
import { createProxyFetch } from "@major-tech/resource-client";
import { tryGetContext } from "hono/context-storage";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.HUBSPOT_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
  getUserJwt: () => tryGetContext()?.req.header("x-major-user-jwt"),
});

const res = await proxyFetch("https://api.hubapi.com/crm/v3/objects/contacts", {
  method: "GET",
  headers: { "Content-Type": "application/json" },
});

const data = await res.json();
```

### Older Next.js apps

Import from `@major-tech/resource-client/next` instead and omit `getUserJwt`: that entry reads `x-major-user-jwt` with `next/headers`. Use it in Server Components, Route Handlers, or Server Actions only — never in client components.

### Limits

- Body and response are clamped to 50 MB by the proxy.
- `X-Major-Timeout-Ms` is clamped to 60_000 (60 s).
- Reserved headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`, `X-Major-*`, `X-Pd-*`) are silently stripped from requests.

---

## MCP Tools

The proxy is also exposed as two generic MCP tools that work across every HTTP-based resource the caller has access to.

- `mcp__resources__http_proxy_invoke` — Make any HTTP method call (GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS). Use for writes or when you specifically need a non-GET verb.
- `mcp__resources__http_proxy_get` — Make a GET request. Read-only — safe in restricted contexts.

**Tool selection:** Prefer a specialized resource tool (e.g. `stripe_list_customers`, `hubspot_get`) when one exists — they have typed args and resource-specific defaults. Fall back to `http_proxy_get` / `http_proxy_invoke` when no specialized tool covers the endpoint.

### Arguments

Both tools take:

| Field         | Type                | Notes                                                                              |
| ------------- | ------------------- | ---------------------------------------------------------------------------------- |
| `description` | `string`            | Brief label (~5 words) shown to the user in chat                                   |
| `resourceId`  | `string`            | UUID of the HTTP-based resource — discover with `mcp__resources__list_resources`   |
| `url`         | `string`            | Full upstream URL (e.g. `https://api.hubapi.com/crm/v3/objects/contacts`)          |
| `headers`     | `Record<string,string>` | Optional. Do NOT set `Authorization` — the proxy injects it.                  |
| `timeoutMs`   | `number`            | Optional. Default 30_000, max 60_000.                                              |

`http_proxy_invoke` additionally takes:

| Field    | Type          | Notes                                                                       |
| -------- | ------------- | --------------------------------------------------------------------------- |
| `method` | `string`      | GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS                                |
| `body`   | `RequestBody` | Optional tagged union: `{ type: "json" \| "form" \| "text" \| "bytes", ... }` |

### Body shape

| Type     | Content-Type                       | `value` shape                                                            |
| -------- | ---------------------------------- | ------------------------------------------------------------------------ |
| `"json"` | `application/json`                 | Any JSON-serializable value                                              |
| `"form"` | `application/x-www-form-urlencoded`| Flat `Record<string, primitive>` — use bracket keys for nesting          |
| `"text"` | `text/plain`                       | `string`                                                                 |
| `"bytes"`| `contentType` (or `application/octet-stream`) | Set `base64` field, not `value`; optional `contentType`           |

### Discovering proxy-compatible resources

`mcp__resources__list_resources` returns each resource with a `proxy` field:

```json
{
  "id": "...",
  "subtype": "stripe",
  "proxy": { "compatible": true, "baseUrls": ["https://api.stripe.com"] }
}
```

Use `proxy.baseUrls` as the allowed host prefixes when constructing the `url` for an invoke call.

---

## Tips

- **`resourceId` must be static.** Pass the resource UUID as a string literal or simple identifier (e.g. an env var read at module scope). Dynamic expressions won't work — the resource won't be tracked against your app.
- **No `Authorization` from your side.** Setting it does nothing (the proxy strips it) and risks leaking a key into a log. Let the proxy inject upstream auth.
- **Per-user OAuth resources** (Gmail, Calendar, Drive) need the user JWT: pass `getUserJwt` (automatic with the `/next` entry in an older Next.js app). Outside a request only the app-level JWT is sent, so per-user calls will fail.
