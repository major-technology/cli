---
name: using-mcp-custom-connector
description: Implements runtime tool calls to a custom (bring-your-own) remote MCP server connector, using the in-session MCP tools for exploration and the generic createMcpClient for app code. Use when doing ANYTHING that touches a custom MCP connector / remote MCP server / mcp_custom resource in any way, load this skill.
---

# Major Platform Resource: Custom MCP Connector (BYO remote MCP server)

A custom MCP connector points at an external remote MCP server you bring. Its tools are defined by that upstream server, not by Major — so there is **no fixed `mcp__resources__mcp_custom_*` tool set**. The available tools, their names, and their argument shapes all come from the upstream server.

## Common: Interacting with Resources

**Security**: Never connect directly to the upstream MCP server. Never put credentials in code. The connector's auth is injected server-side using the resolved shared or per-user credential.

**Two ways to interact with a custom MCP connector:**

1. **In-session MCP tools** (direct, no code needed): the connector is mounted as its own MCP server, so its tools are callable during the build as `mcp__<slug>__<toolName>`. Use `mcp__resources__list_resources` to find the connector and its `resourceId`. Call these tools to discover what the upstream server exposes and to test behavior.
2. **Generic MCP client** (for app code): import `createMcpClient` from `@major-tech/resource-client/next` and call `.callTool()`. There is **no per-resource generation step** — it's one client reused for any MCP connector; you just pass the `resourceId`.

**CRITICAL: Do NOT guess tool names or argument shapes.** They are defined by the upstream MCP server, not by Major or by the client — `callTool(name, args)` is a single generic method with no per-tool typed methods. Discover the real tool names and arg shapes from the in-session `mcp__<slug>__*` tools before wiring them into app code.

**Server-side only**: the app JWT must never reach the browser, so call `createMcpClient` from Next server code (Server Components, Server Actions, Route Handlers).

**App context required**: `callTool()` needs the app's `baseUrl` / `applicationId` / JWT, which are injected into the deployed app's environment. In a coding session, reach the connector through the `mcp__<slug>__*` tools instead.

**Error handling**: unlike other resource clients, `callTool()` **throws** `ResourceInvokeError` on a transport failure or an error response — there is **no `result.ok` envelope** to check. Wrap calls in `try/catch`. A tool-level failure that the upstream reports successfully comes back as a normal result with `isError: true`.

---

## MCP Tools (in-session)

There is no static tool list. The connector's upstream tools are mounted as `mcp__<slug>__<toolName>`, where `<slug>` is the connector's mount slug. Examples depend entirely on the upstream server (e.g. `mcp__acme__search_tickets`, `mcp__acme__create_ticket`). Use `mcp__resources__list_resources` to find the connector, then call its mounted tools to learn the exact names and arguments.

## TypeScript Client

```typescript
import { createMcpClient } from "@major-tech/resource-client/next";

// Config comes from the deployed app's injected env; resourceId is the connector.
const mcp = createMcpClient({
	baseUrl: process.env.MAJOR_API_BASE_URL!,
	applicationId: process.env.APPLICATION_ID!,
	resourceId: "<connector resourceId>",
	majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

// callTool<T>(tool, args?) — `tool` is the upstream MCP tool name; `args` is
// forwarded verbatim. Throws ResourceInvokeError on transport/server error.
try {
	const result = await mcp.callTool<{ tickets: Ticket[] }>(
		"search_tickets",
		{ customerId, limit: 20 },
	);

	if (result.isError) {
		// the tool itself reported an error — detail is in result.content
		console.error(result.content);
		return;
	}

	const data = result.structuredContent; // typed as { tickets: Ticket[] } | undefined
	// or read result.content (text/other blocks) when the tool returns no structured payload
} catch (err) {
	// ResourceInvokeError: transport failure or an error envelope from the proxy
}
```

## Tips

- **mcp_custom connectors are callable from app code.** They used to be reachable only through in-session MCP tools; app code can now call their tools at runtime via `createMcpClient(...).callTool()`. Don't tell the user a custom MCP connector "can't be used from the app."
- **Result shape** mirrors the MCP `CallToolResult`: read `result.structuredContent` (typed via the `<T>` you pass) for structured payloads, or `result.content` for unstructured blocks; check `result.isError` for tool-level failures.
- **Args are forwarded verbatim** to the upstream tool — match the upstream server's expected schema exactly.
- **Auth is injected server-side** using the resolved shared or per-user credential; never set auth headers yourself.
