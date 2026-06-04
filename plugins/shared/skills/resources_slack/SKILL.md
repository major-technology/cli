---
name: using-slack-connector
description: Implements Slack messaging, channel operations, and Web API calls using generated clients and MCP tools. Use when doing ANYTHING that touches Slack in any way, load this skill.
---

# Major Platform Resource: Slack

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Three ways to interact with Slack:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).
3. **HTTP proxy** (Next.js apps): Use `createProxyFetch` from `@major-tech/resource-client/next` to call the Slack API directly with automatic auth injection. See [using-http-proxy](../http-proxy/SKILL.md) for setup and usage — preferred when you need to hit endpoints not covered by MCP tools or the typed client, or when using an official SDK that accepts a custom `fetch`.

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## CRITICAL: Channel Access Verification

Before sending messages, posting files, or reading history from any channel, you MUST verify the bot has access to that channel. **Never attempt to post to a channel without confirming access first.**

**Required workflow:**

1. Call `mcp__resources__slack_list_channels` to get the list of channels the bot can see.
2. Check if the target channel appears in the results.
3. **If the channel is NOT in the list:** Tell the user that the bot does not currently have access to that channel. Ask them to invite the bot by going to the channel and @mentioning **@Major Slack Integration**. Once the user confirms they have done this, call `mcp__resources__slack_list_channels` again to verify the channel now appears.
4. **Only after the channel is confirmed visible** in the list may you proceed with sending messages, reading history, or any other channel operation.

**Do NOT skip this check.** Do NOT assume the bot has access to a channel just because the user mentioned it by name.

---

## MCP Tools

- `mcp__resources__slack_call` — Call any Slack Web API method. Args: `resourceId`, `method`, `body?`
- `mcp__resources__slack_list_channels` — List channels in the workspace. Args: `resourceId`, `limit?`
- `mcp__resources__slack_post_message` — Post a message to a channel. Args: `resourceId`, `channel`, `text`, `blocks?`
- `mcp__resources__slack_get_history` — Get message history from a channel. Args: `resourceId`, `channel`, `limit?`

## TypeScript Client

```typescript
import { slackClient } from "./clients";

// invoke(method, invocationKey, options?)
// The `method` parameter is the Slack API method name
const result = await slackClient.invoke("chat.postMessage", "post-update", {
	body: { channel: "C0123456", text: "Hello from the app!" },
});

// List channels
await slackClient.invoke("conversations.list", "list-channels", {
	body: { limit: 100 },
});

// getUploadURL(filename, length, invocationKey, options?)
// completeUpload(files, channelId, invocationKey, options?)
// See "File Upload" section below for usage
```

## File Upload

Slack uses a 3-step flow for uploading files/images to channels:

```typescript
import { slackClient } from "./clients";

// Step 1: Get a pre-signed upload URL
const urlResult = await slackClient.getUploadURL(
	"chart.png", // filename with extension
	fileBytes.length, // file size in bytes
	"get-upload-url",
);
if (!urlResult.ok) throw new Error(urlResult.error.message);
const { upload_url, file_id } = urlResult.result.body.value;

// Step 2: Upload the file binary to the pre-signed URL
await fetch(upload_url, {
	method: "POST",
	headers: { "Content-Type": "application/octet-stream" },
	body: fileBytes,
});

// Step 3: Complete the upload and share to a channel
const completeResult = await slackClient.completeUpload(
	[{ id: file_id, title: "Weekly Chart" }],
	"C0123456", // channel ID
	"complete-upload",
	{ initialComment: "Here's this week's chart" },
);
```

- The upload URL from step 1 is temporary — complete all 3 steps without delay
- Step 2 is a direct HTTP POST (no auth needed, the URL is pre-signed)
- You can upload multiple files by calling step 1+2 for each, then passing all file IDs to a single step 3
- Use `threadTs` in step 3 options to upload into a thread
- Requires `files:write` OAuth scope (included in the "Read & Write" preset)

## Tips

- The `method` param is the **Slack API method name** (e.g., `chat.postMessage`, `conversations.list`, `users.list`)
- For the TypeScript client, all parameters go in the `body` option — Slack's Web API uses POST with JSON body
- Check [Slack API methods list](https://api.slack.com/methods) for available methods and their parameters

**Docs**: [Slack API Reference](https://api.slack.com/methods)
