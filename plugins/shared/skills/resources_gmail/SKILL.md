---
name: using-gmail-connector
description: Implements Gmail email reading, searching, and sending using generated clients and MCP tools. Use when doing ANYTHING that touches Gmail or email in any way, load this skill.
---

# Major Platform Resource: Gmail

## Setting Up a Gmail Connector

Gmail requires OAuth authentication before use.

### When the user asks you to set up Gmail or connect their email:

1. Call `mcp__resource-setup__request-resource-setup` with `subtype: "gmail"` — this prompts the user to authenticate with Google
2. Once setup completes, the resource is ready to use

---

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "Search recent emails", "Send meeting invite"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-recent-emails"`, never dynamic values like `` `${date}-emails` ``.

---

## MCP Tools

- `mcp__resources__gmail_list_messages` — Search and list emails. Args: `resourceId`, `q?` (Gmail search syntax), `maxResults?`, `pageToken?`. **Returns only message IDs and thread IDs** — always follow up with `gmail_get_message` for headers or content.
- `mcp__resources__gmail_get_message` — Get one message by ID. Args: `resourceId`, `messageId`, `format?` (default: `"full"`). Use `format="metadata"` when you only need headers, and the default `format="full"` for message content — it returns a **normalized** response (see below). Never reach for `gmail_invoke` to read an ordinary message.
- `mcp__resources__gmail_send_message` — Send a plain-text email. Args: `resourceId`, `to`, `subject`, `body`, `cc?`, `bcc?`. Requires the `readwrite` scope preset.
- `mcp__resources__gmail_list_labels` — List all Gmail labels (inbox, sent, custom labels). Args: `resourceId`
- `mcp__resources__gmail_invoke` — Escape hatch for Gmail API v1 operations the typed tools don't cover (complete threads, drafts, label modification, attachment bytes). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`. Returns the **unmodified** Gmail response, so it is not bounded — see the fallback section below.

### Reading messages

Standard flow — two steps, no raw MIME, no manual decoding:

1. `gmail_list_messages` → message IDs.
2. `gmail_get_message` per ID → `format="metadata"` for headers only, or the default `format="full"` for content.

`format="full"` is normalized into an agent-friendly, size-bounded object at `body.value`:

| Field (full path in the tool result) | Meaning |
| --- | --- |
| `body.value.headers` | Selected headers only: `from`, `to`, `cc`, `bcc`, `subject`, `date`, `reply-to`, `in-reply-to`, `message-id`. Each is `{ name, value }`. |
| `body.value.body.text` | The **already-decoded** message text. Prefers `text/plain`, falling back to `text/html`. |
| `body.value.body.mimeType` | Which of the two the text came from (`text/plain` or `text/html`). |
| `body.value.body.truncated` | `true` when the text was cut to fit the budget. |
| `body.value.body.originalChars` | Character count of the full decoded text before truncation, so you can tell how much you're missing. |
| `body.value.attachments` | Bounded metadata only — `filename`, `mimeType`, `size`, `attachmentId`. **Never base64 payload bytes.** |
| `body.value.attachmentsTruncated` | `true` when attachments were dropped to fit the budget. |

(`body.value` is the message; `body.value.body` is its decoded content — the outer `body` is the HTTP response envelope.)

The text is capped at 16,000 characters and the whole formatted response at roughly 20KB; when the response would still be too large, the body text is trimmed further and then attachments, labels, and headers are shed. Read the `truncated` / `originalChars` / `attachmentsTruncated` flags rather than assuming you got everything.

```jsonc
// gmail_get_message(resourceId, messageId, format: "full") →
{
  "kind": "api",
  "status": 200,
  "body": {
    "kind": "json",
    "value": {
      "id": "m1",
      "threadId": "t1",
      "labelIds": ["INBOX", "UNREAD"],
      "snippet": "hi",
      "headers": [
        { "name": "From", "value": "a@b.com" },
        { "name": "Subject", "value": "Hello" }
      ],
      "body": { "mimeType": "text/plain", "text": "Hello", "truncated": false, "originalChars": 5 },
      "attachments": [
        { "filename": "a.pdf", "mimeType": "application/pdf", "size": 10, "attachmentId": "att1" }
      ],
      "attachmentsTruncated": false
    }
  }
}
```

Because the text arrives decoded, **do not** base64-decode it, and do not write the response to a file to `Read`/`grep`/parse it back — read `body.value.body.text` directly from the tool result.

Only `format="full"` is normalized. `metadata`, `minimal`, and `raw` pass through as the upstream Gmail shape, so `metadata` headers stay under `body.value.payload.headers` and `raw` still returns base64 (`raw` is rarely what you want — prefer `full`). Gmail error responses and non-2xx statuses are passed through untouched, so keep checking `status`.

### When to fall back to `gmail_invoke`

Reserve it for operations the typed tools don't cover — most commonly fetching a **complete thread**:

```jsonc
gmail_invoke({
  resourceId,
  method: "GET",
  path: "users/me/threads/THREAD_ID",
  query: { "format": ["metadata"] }   // values are ARRAYS of strings
})
```

`query` is `map[string][]string`: every value must be an array, so it's `{"format": ["metadata"]}`, never `{"format": "metadata"}`. Since `gmail_invoke` returns the raw Gmail response, a thread fetched with `format="full"` carries base64 MIME parts for every message and can be enormous — prefer `format=["metadata"]` to enumerate the thread, then `gmail_get_message` per message ID for bounded content.

### If any Gmail tool result comes back as a file reference

Regardless of which tool triggered it, do not `Read` the file into context. Use `jq` to pull out only what you need:

```bash
# Headers only, across every message in a thread dump
jq '.messages[].payload.headers' /path/to/response.json

# Just the plain-text body part of one message, still base64 (decode after)
jq -r '.payload.parts[] | select(.mimeType == "text/plain") | .body.data' /path/to/response.json

# Count messages / list IDs without loading bodies
jq '.messages | map(.id)' /path/to/response.json
```

Filter with `jq` first, decode base64 (`| base64 -d`) only on the small slice you actually need, and never pipe the full file through `cat`/`Read`.

## TypeScript Client

```typescript
import { gmailClient } from "./clients";

// invoke(method, path, invocationKey, options?)
// All paths are relative to https://gmail.googleapis.com/gmail/v1/

// Search for recent emails
const result = await gmailClient.invoke("GET", "users/me/messages", "search-emails", {
	query: { q: "is:unread from:team@company.com", maxResults: "10" },
});
if (result.ok && result.result.status === 200 && result.result.body.kind === "json") {
	const messages = result.result.body.value.messages;
}

// Get a specific message. The client is a thin wrapper over the Gmail API, so this
// returns the RAW Gmail shape (base64 MIME parts under body.value.payload) — the
// normalization and size bounds described above apply to the gmail_get_message MCP
// tool, not to gmailClient.invoke. In app code, decode payload parts yourself, or
// request format=metadata when headers are enough.
const msgResult = await gmailClient.invoke("GET", "users/me/messages/MSG_ID", "get-message", {
	query: { format: "metadata" },
});
```

## Tips

- **All paths are relative to `https://gmail.googleapis.com/gmail/v1/`** — e.g. use `users/me/messages`, not the full URL.
- **Gmail search syntax**: `from:user@example.com`, `subject:meeting`, `after:2024/01/01`, `is:unread`, `has:attachment`, `label:INBOX`. Combine with spaces (AND) or `OR`.
- **Message format options**: `full` (default — normalized, bounded, decoded text), `metadata` (headers only), `minimal` (IDs only), `raw` (RFC 2822, base64). Only `full` is normalized.
- **Two-step read pattern**: `list_messages` returns only IDs → `get_message` with `format="metadata"` for headers or the default `format="full"` for content. Use `gmail_invoke` only for what the typed tools don't cover, e.g. complete threads.
- **File-reference responses**: if any Gmail tool result comes back as a file reference instead of inline JSON, use `jq` to extract the fields you need (see above) rather than reading the whole file.
- **Pagination**: Check `nextPageToken` in the response and pass it as `pageToken` to get the next page.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`
- **Scope presets**: "readonly" (read/search only) or "readwrite" (read/search + send). Send operations fail with 403 on readonly.
- **Common paths**: `users/me/messages` (list/search), `users/me/messages/{id}` (get), `users/me/messages/send` (send), `users/me/labels` (list labels), `users/me/threads` (list threads)

**Docs**: [Gmail API Reference](https://developers.google.com/gmail/api/reference/rest)
