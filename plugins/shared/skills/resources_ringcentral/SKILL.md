---
name: using-ringcentral-connector
description: Implements RingCentral API access (call logs, messages, SMS, extensions) through the Major HTTP proxy. Use when doing ANYTHING that touches RingCentral in any way, load this skill.
---

# Major Platform Resource: RingCentral

RingCentral is a proxy-only resource — every call goes through Major's HTTP proxy. There is **no typed `ringcentral_*` MCP tool and no generated TypeScript client**. The proxy resolves the tenant host (`<instance>.ringcentral.com`) at request time from the connected account, so you never hardcode or pass the instance host — always pass a **leading-slash relative path** like `/restapi/v1.0/account/~/call-log`.

**Security**: Never set the `Authorization` header — the proxy injects it. Reserved headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`, `X-Major-*`) are stripped on the way out.

---

## MCP Tools

Use the generic HTTP proxy tools — there are no RingCentral-specific MCP tools.

- `mcp__resources__http_proxy_get` — Read-only GET. Args: `resourceId`, `url` (leading-slash path), `headers?`, `timeoutMs?`
- `mcp__resources__http_proxy_invoke` — Any HTTP method. Args: `resourceId`, `method`, `url` (leading-slash path), `headers?`, `body?`, `timeoutMs?`

See [using-http-proxy](../http-proxy/SKILL.md) for the full proxy reference.

## HTTP Proxy via `createProxyFetch` (Next.js)

For app code, drop the generic `createProxyFetch` into any SDK that takes a custom `fetch`, or use it as a plain `fetch` wrapper. Pass **leading-slash paths** instead of a full URL — the proxy resolves the upstream host.

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.RINGCENTRAL_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

// List recent call log entries
const res = await proxyFetch("/restapi/v1.0/account/~/call-log?perPage=25&dateFrom=2024-01-01T00:00:00Z");
if (!res.ok) throw new Error(`RingCentral ${res.status}: ${await res.text()}`);
const { records } = await res.json();

// List extensions
const ext = await proxyFetch("/restapi/v1.0/account/~/extension?perPage=100");

// Send an SMS
const sms = await proxyFetch("/restapi/v1.0/account/~/extension/~/sms", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    from: { phoneNumber: "+15551234567" },
    to: [{ phoneNumber: "+15559876543" }],
    text: "Hello from Major!",
  }),
});

// List messages from the message store
await proxyFetch("/restapi/v1.0/account/~/extension/~/message-store?messageType=SMS&perPage=25");
```

**Rules:**

- `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they're set, don't ask the user to provide them. `RINGCENTRAL_RESOURCE_ID` is the UUID of the connected RingCentral resource.
- **Always pass a leading-slash path.** Hardcoding `https://<instance>.ringcentral.com/...` will fail — the instance host isn't known to app code.
- **`resourceId` must be a static string literal** — the query extractor only tracks proxy calls when the resource id is known at build time.
- Server-side only (Server Components, Route Handlers, Server Actions). The `/next` subpath uses `next/headers`.
- Do NOT set the `Authorization` header — the proxy injects upstream auth.

## Common RingCentral endpoints

| What                 | Method | Path                                                                  |
| -------------------- | ------ | --------------------------------------------------------------------- |
| Current extension    | GET    | `/restapi/v1.0/account/~/extension/~`                                 |
| List extensions      | GET    | `/restapi/v1.0/account/~/extension`                                   |
| Get one extension    | GET    | `/restapi/v1.0/account/~/extension/{extensionId}`                     |
| List call log        | GET    | `/restapi/v1.0/account/~/call-log`                                    |
| Get one call record  | GET    | `/restapi/v1.0/account/~/call-log/{callRecordId}`                     |
| List messages        | GET    | `/restapi/v1.0/account/~/extension/~/message-store`                   |
| Send SMS             | POST   | `/restapi/v1.0/account/~/extension/~/sms`                             |

The `~` token means "the current account / authenticated extension" — RingCentral resolves it from the connected credential, so you don't substitute an id.

## Tips

- **`~` is RingCentral's self-reference.** Use `account/~` for the connected account and `extension/~` for the authenticated extension; pass a concrete id only when targeting a different extension.
- **Call log & message filters are query params**: `dateFrom`, `dateTo`, `direction` (`Inbound`/`Outbound`), `type` (`Voice`/`Fax`), `messageType` (`SMS`/`Fax`/`VoiceMail`/`Pager`), `perPage`, `page`.
- **SMS bodies are structured**: `{ from: { phoneNumber }, to: [{ phoneNumber }], text }`. Numbers must be E.164 (`+1...`).
- **Pagination** is page-based: `perPage` + `page`; responses carry `paging` and `navigation` blocks.
- **Non-2xx upstream statuses are passed through** — always inspect `res.ok` / `res.status` before treating the body as success.
- **Rate limits**: RingCentral groups endpoints into Light/Medium/Heavy buckets with per-minute limits; expect `429` with a `Retry-After` header under load.

**Docs**: [RingCentral REST API Reference](https://developers.ringcentral.com/api-reference)
