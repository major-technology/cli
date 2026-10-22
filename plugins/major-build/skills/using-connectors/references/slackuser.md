# Major Platform Resource: Slack (Personal)

Slack (Personal) acts in Slack **as the user who connected it**, with that user's own Slack token. It sees exactly the channels, private channels, DMs, and group DMs that user is in, can search the whole workspace (`search.messages`), and posts, edits, and reacts under the user's own name. It never uses the org's "Major" bot. For posting as the bot, use the org Slack connector ([slack.md](slack.md)).

There is **no generated TypeScript client**. Use the typed MCP tools below, or the HTTP proxy for any other Slack Web API method.

**Security**: Never set the `Authorization` header — the proxy injects the user's token. Reserved headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`, `X-Major-*`) are stripped on the way out.

---

## Setting Up a Slack (Personal) Connector

1. Call `mcp__plugin_major_major__request_resource_setup` with `connectorId: "slackuser"` — this prompts the user to sign in to Slack.
2. Once setup completes, the resource is ready to use. Each user connects their own Slack account.

A Slack API error of `missing_scope` means the user connected before write access existed: ask them to reconnect Slack (Personal), then retry.

---

## Posting as the user

Everything this connector writes appears in Slack **as the user**, not as a bot. Write tools wait for the user's approval before they run.

- Only post, edit, or react when the user asked for it. Never post on their behalf proactively.
- Show the user the exact text and destination before posting anything long or to a wide channel.
- `update_message` can only edit messages the user themselves posted.

---

## MCP Tools

Read tools (run without approval):

- `mcp__resources__slackuser_search_messages` — Search messages across the whole workspace. Args: `resourceId`, `query` (Slack search syntax, e.g. `from:@jane in:#general budget after:2026-01-01`), `count?` (max 100), `page?`, `sort?` (`score` | `timestamp`), `sortDir?`. Prefer this for any workspace-wide lookup.
- `mcp__resources__slackuser_search_channels` — Find channels the user can see by name. Args: `resourceId`, `query`, `limit?`, `types?`
- `mcp__resources__slackuser_list_channels` — List channels the user can see. Args: `resourceId`, `limit?`, `cursor?`, `types?` (`public_channel`, `private_channel`, `mpim`, `im`; default `public_channel`)
- `mcp__resources__slackuser_get_history` — Message history of a channel or DM the user is in. Args: `resourceId`, `channel`, `limit?`, `cursor?`
- `mcp__resources__slackuser_search_history` — Text search within one channel's history, paginated by `nextCursor`/`hasMore`. Args: `resourceId`, `channel`, `query`, `cursor?`, `limit?`

Write tools (need approval):

- `mcp__resources__slackuser_post_message` — Post as the user. Args: `resourceId`, `channel` (channel or DM ID), `text` (Slack mrkdwn), `threadTs?` (parent message `ts` to reply in its thread)
- `mcp__resources__slackuser_update_message` — Edit one of the user's own messages. Args: `resourceId`, `channel`, `ts`, `text`
- `mcp__resources__slackuser_add_reaction` — React to a message. Args: `resourceId`, `channel`, `ts`, `name` (emoji name without colons, e.g. `thumbsup`)
- `mcp__resources__slackuser_open_dm` — Open (or find) a DM or group DM and return its channel ID for `post_message`. Args: `resourceId`, `users` (Slack user IDs; one for a DM, up to eight for a group DM)

To DM someone: find their user ID (`users.lookupByEmail` or `users.list` through the proxy), call `open_dm`, then `post_message` to the returned channel ID.

Anything else (deleting a message, file uploads, user lookups, removing reactions):

- `mcp__resources__http_proxy_get` / `mcp__resources__http_proxy_invoke` — Args: `resourceId`, `url` (leading-slash Slack method path, e.g. `/users.lookupByEmail?email=jane@example.com`), `method` (invoke only), `headers?`, `body?` (invoke only), `timeoutMs?`. `http_proxy_get` is read-only; `http_proxy_invoke` needs approval. See [using-http-proxy](http-proxy.md).

## HTTP Proxy via `createProxyFetch` (Next.js)

For app code, use `createProxyFetch` with **leading-slash Slack method paths** — they resolve against `https://slack.com/api`.

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.SLACKUSER_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

// Search as the signed-in user
const search = await proxyFetch("/search.messages?query=" + encodeURIComponent("from:@jane budget"));
const searchBody = await search.json();
if (!searchBody.ok) throw new Error(`Slack: ${searchBody.error}`);

// Post as the signed-in user
const post = await proxyFetch("/chat.postMessage", {
  method: "POST",
  headers: { "Content-Type": "application/json; charset=utf-8" },
  body: JSON.stringify({ channel: "C0123456", text: "Posted from my app" }),
});
const postBody = await post.json();
if (!postBody.ok) throw new Error(`Slack: ${postBody.error}`);
```

**Rules:**

- Slack answers most errors with HTTP 200 and `{ ok: false, error }` — always check `ok` in the body, not just `res.ok`.
- `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they're set. `SLACKUSER_RESOURCE_ID` is the UUID of the connected Slack (Personal) resource.
- **`resourceId` must be a static string literal** — the query extractor only tracks proxy calls when the resource id is known at build time.
- Server-side only (Server Components, Route Handlers, Server Actions). The `/next` subpath uses `next/headers`, which forwards the signed-in user so each user acts as themselves.

## Common methods

| What                     | Method | Path                                                      |
| ------------------------ | ------ | --------------------------------------------------------- |
| Delete own message       | POST   | `/chat.delete` with `{ channel, ts }`                     |
| Remove a reaction        | POST   | `/reactions.remove` with `{ channel, timestamp, name }`   |
| Look up a user by email  | GET    | `/users.lookupByEmail?email={email}`                      |
| Get a thread's replies   | GET    | `/conversations.replies?channel={id}&ts={parentTs}`       |
| File info                | GET    | `/files.info?file={fileId}`                               |
| Upload a file (step 1)   | GET    | `/files.getUploadURLExternal?filename={name}&length={n}`  |
| Upload a file (step 3)   | POST   | `/files.completeUploadExternal` with `{ files, channel_id }` |

File upload is three steps: get an upload URL, `POST` the bytes to that pre-signed URL directly (not through the proxy), then complete the upload to share it in a channel. File links (`url_private`) are fetched through the proxy with their full `https://files.slack.com/...` URL.

## Tips

- See the rate-limit guidance in [slack.md](slack.md) — it applies here too: query serially, use large `limit`s, and stop retrying into 429s.
- If a history or search response is written to a file instead of returned inline, read it with `jq` (e.g. `jq '.messages[] | {user, text, ts}' file.json`).

**Docs**: [Slack API Reference](https://api.slack.com/methods)
