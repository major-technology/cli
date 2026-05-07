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

- `mcp__resources__gmail_list_messages` — Search and list emails. Args: `resourceId`, `query?` (Gmail search syntax), `maxResults?`, `pageToken?`. **Returns only message IDs and thread IDs** — always follow up with `gmail_get_message` for content.
- `mcp__resources__gmail_get_message` — Get a specific email by ID with full content. Args: `resourceId`, `messageId`, `format?` (default: "full")
- `mcp__resources__gmail_send_message` — Send a plain-text email. Args: `resourceId`, `to`, `subject`, `body`, `cc?`, `bcc?`. Requires the `readwrite` scope preset.
- `mcp__resources__gmail_list_labels` — List all Gmail labels (inbox, sent, custom labels). Args: `resourceId`
- `mcp__resources__gmail_invoke` — Make any HTTP request to the Gmail API v1 (for operations not covered by other tools). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

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

// Get a specific message
const msgResult = await gmailClient.invoke("GET", "users/me/messages/MSG_ID", "get-message", {
	query: { format: "full" },
});
```

## Tips

- **All paths are relative to `https://gmail.googleapis.com/gmail/v1/`** — e.g. use `users/me/messages`, not the full URL.
- **Gmail search syntax**: `from:user@example.com`, `subject:meeting`, `after:2024/01/01`, `is:unread`, `has:attachment`, `label:INBOX`. Combine with spaces (AND) or `OR`.
- **Message format options**: `full` (headers + parsed body), `metadata` (headers only), `minimal` (IDs only), `raw` (RFC 2822 encoded).
- **Two-step read pattern**: `list_messages` returns only IDs → use `get_message` to fetch full content.
- **Pagination**: Check `nextPageToken` in the response and pass it as `pageToken` to get the next page.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`
- **Scope presets**: "readonly" (read/search only) or "readwrite" (read/search + send). Send operations fail with 403 on readonly.
- **Common paths**: `users/me/messages` (list/search), `users/me/messages/{id}` (get), `users/me/messages/send` (send), `users/me/labels` (list labels), `users/me/threads` (list threads)

**Docs**: [Gmail API Reference](https://developers.google.com/gmail/api/reference/rest)
