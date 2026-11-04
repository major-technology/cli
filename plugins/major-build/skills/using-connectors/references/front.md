# Major Resource: Front

Front (shared inbox / customer communication) is reached through Front's own MCP server. The connector is mounted as an MCP server, so its tools are callable as `mcp__<slug>__<toolName>`. Major exposes **no** `mcp__resources__front_*` tools and **no** HTTP proxy for Front.

## Common: Interacting with Resources

**Security**: Never connect directly to Front's MCP server or REST API. Never put credentials in code. The OAuth token is injected server-side.

**Per-user**: a Front token belongs to the teammate who connected it. Every call runs as that teammate, with that teammate's Front permissions, and is attributed to them in Front (sent messages, comments, assignments). On a `per_user` connector each user connects their own Front account; if a call returns `USER_OAUTH_REQUIRED`, call `mcp__plugin_major_major__setup_user_oauth` with the connector's resource id.

**Two ways to interact with Front:**

1. **In-session MCP tools** (direct, no code needed): call `mcp__<slug>__<toolName>` through `mcp__plugin_major_major__execute_resource_tool`. Use `mcp__resources__list_resources` to find the connector, its `resourceId`, and its slug.
2. **Generic MCP client** (for app code): `createMcpClient` from `@major-tech/resource-client/next`, called with the connector's `resourceId`. There is no per-resource generation step.

**CRITICAL: Do NOT guess argument shapes.** They are defined by Front's MCP server. Call the in-session tool (or search the tool catalog) to learn a tool's arguments before wiring it into app code.

**Server-side only**: call `createMcpClient` from Next server code (Server Components, Server Actions, Route Handlers); the app JWT must never reach the browser.

**Error handling**: `callTool()` **throws** `ResourceInvokeError` on a transport failure or an error response; there is no `result.ok` envelope. A tool-level failure comes back as a normal result with `isError: true`.

---

## MCP Tools (in-session)

Front's server exposes these tools. Write tools change what teammates and customers see; confirm with the user before running them.

| Area | Tools |
| --- | --- |
| Conversations | `search_conversations`, `read_conversation`, `create_draft`, `update_draft`, `delete_draft`, `send_message`, `update_conversation_status`, `assign_conversation`, `move_conversation` |
| Messages | `read_message`, `get_attachment` |
| Message templates | `search_message_templates` |
| Comments | `add_comment` |
| Tasks | `create_task`, `update_task` |
| Tags | `list_tags`, `tag_conversation` |
| Contacts and accounts | `search_contacts`, `read_contact`, `search_accounts`, `read_account` |
| Organization | `get_my_identity`, `list_inboxes`, `list_channels`, `list_teammates`, `list_teams`, `list_statuses`, `list_drafts` |

`send_message` sends to a customer and cannot be undone. Prefer `create_draft` unless the user asked to send.

## TypeScript Client

```typescript
import { createMcpClient } from "@major-tech/resource-client/next";

const front = createMcpClient({
	baseUrl: process.env.MAJOR_API_BASE_URL!,
	applicationId: process.env.APPLICATION_ID!,
	resourceId: "<front resourceId>",
	majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

try {
	// Arguments are Front's; read them from the in-session tool first.
	const result = await front.callTool("search_conversations", { query: "refund" });

	if (result.isError) {
		console.error(result.content);
		return;
	}

	const data = result.structuredContent ?? result.content;
} catch (err) {
	// ResourceInvokeError: transport failure or an error envelope from the proxy
}
```

## Tips

- **Start with `get_my_identity`** to confirm which teammate the connection acts as, and `list_inboxes` / `list_teammates` / `list_tags` to resolve names to the ids other tools take.
- **Visibility is the teammate's**: a conversation missing from `search_conversations` may simply be outside the connected teammate's inboxes, not absent from Front.
- **Permissions come from Front**: the Front app's resource permissions and the teammate's role bound what any tool can do. A forbidden error means Front denied it; reconnecting does not widen it.
