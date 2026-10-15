# Major Platform Resource: Outlook Mail

Outlook Mail is a proxy-only resource backed by Microsoft Graph — every call goes through Major's HTTP proxy. There is **no typed `outlook_*` MCP tool and no generated TypeScript client**. Use a **leading-slash relative path** under `https://graph.microsoft.com`, for example `/v1.0/me/messages?$top=25`. For calendar events, use the separate Outlook Calendar connector ([outlookcalendar.md](outlookcalendar.md)).

**Security**: Never set the `Authorization` header — the proxy injects it. Reserved headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`, `X-Major-*`) are stripped on the way out.

---

## Setting Up an Outlook Mail Connector

Outlook Mail requires Microsoft OAuth before use.

1. Call `mcp__plugin_major_major__request_resource_setup` with `connectorId: "outlook"` — this prompts the user to sign in with Microsoft.
2. Once setup completes, the resource is ready to use.

The connection has a scope preset. `readonly` can read and search mail; `readwrite` can also create, update, delete, and send mail. A `readonly` connection returns `403` on writes. A Microsoft 365 admin may need to approve mail access for the organization.

---

## MCP Tools

Use the generic HTTP proxy tools — there are no Outlook-specific MCP tools.

- `mcp__resources__http_proxy_get` — Read-only GET. Args: `resourceId`, `url` (leading-slash path), `headers?`, `timeoutMs?`
- `mcp__resources__http_proxy_invoke` — Any HTTP method. Args: `resourceId`, `method`, `url` (leading-slash path), `headers?`, `body?`, `timeoutMs?`

See [using-http-proxy](http-proxy.md) for the full proxy reference.

## HTTP Proxy via `createProxyFetch` (Next.js)

For app code, use the generic `createProxyFetch` as a plain `fetch` wrapper. Pass **leading-slash paths** instead of a full URL.

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.OUTLOOK_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

// Latest inbox messages
const res = await proxyFetch(
  "/v1.0/me/mailFolders/inbox/messages?$top=25&$select=id,subject,from,receivedDateTime,isRead&$orderby=receivedDateTime desc",
);
if (!res.ok) throw new Error(`Outlook ${res.status}: ${await res.text()}`);
const { value: messages } = await res.json();

// Send mail
const sent = await proxyFetch("/v1.0/me/sendMail", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    message: {
      subject: "Hello from Major",
      body: { contentType: "Text", content: "Hi there" },
      toRecipients: [{ emailAddress: { address: "teammate@example.com" } }],
    },
    saveToSentItems: true,
  }),
});
```

**Rules:**

- `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they're set, don't ask the user to provide them. `OUTLOOK_RESOURCE_ID` is the UUID of the connected Outlook resource.
- **Always pass a leading-slash path.** Do not hardcode `https://graph.microsoft.com/...`.
- **`resourceId` must be a static string literal** — the query extractor only tracks proxy calls when the resource id is known at build time.
- Server-side only (Server Components, Route Handlers, Server Actions). The `/next` subpath uses `next/headers`.
- Do NOT set the `Authorization` header — the proxy injects upstream auth.

## Common Outlook Mail endpoints

| What                    | Method | Path                                                               |
| ----------------------- | ------ | ------------------------------------------------------------------ |
| List messages           | GET    | `/v1.0/me/messages`                                                |
| List a folder's mail    | GET    | `/v1.0/me/mailFolders/{folderId or well-known name}/messages`      |
| Search messages         | GET    | `/v1.0/me/messages?$search="{terms}"`                              |
| Get a message           | GET    | `/v1.0/me/messages/{messageId}`                                    |
| Send mail               | POST   | `/v1.0/me/sendMail`                                                |
| Reply                   | POST   | `/v1.0/me/messages/{messageId}/reply`                              |
| Reply all               | POST   | `/v1.0/me/messages/{messageId}/replyAll`                           |
| Move to a folder        | POST   | `/v1.0/me/messages/{messageId}/move`                               |
| List folders            | GET    | `/v1.0/me/mailFolders`                                             |
| Shared mailbox messages | GET    | `/v1.0/users/{mailboxAddressOrId}/messages`                        |

Well-known folder names include `inbox`, `sentitems`, `drafts`, `deleteditems`, and `archive`. Reply and reply-all take `{ comment, message? }`. Move takes `{ destinationId }`.

## Tips

- **Search with `$search`**: wrap the terms in double quotes, for example `$search="from:alice invoice"`. `$search` cannot be combined with `$orderby`, and results are ranked, not date-sorted. To filter and sort, use `$filter` with `$orderby` instead (for example `$filter=receivedDateTime ge 2026-01-01T00:00:00Z`).
- **Trim responses** with `$select`, and set the page size with `$top`.
- **Pagination** uses `@odata.nextLink` in the response. Request the next page by passing its path and query as the next `url` (strip `https://graph.microsoft.com`).
- **Message bodies** come back as HTML by default. Send `Prefer: outlook.body-content-type="text"` to get plain text.
- **Shared mailboxes** are reached with `/v1.0/users/{mailbox}/...` in place of `/v1.0/me/...`. The connected account needs access to that mailbox, and the connection needs the shared-mail scopes (granted by the `readonly` and `readwrite` presets).
- **`readonly` connections return `403` on writes** (send, reply, move, delete). Ask the user to reconnect with the `readwrite` preset.
- **`sendMail` returns `202` with an empty body** on success. Do not parse JSON from it.
- **Non-2xx upstream statuses are passed through** — always inspect `res.ok` / `res.status` before treating the body as success.

**Docs**: [Microsoft Graph mail API](https://learn.microsoft.com/en-us/graph/api/resources/mail-api-overview)
