---
name: using-notion-connector
description: Implements Notion API interactions for pages, databases, blocks, users, and search using generated clients and MCP tools. Use when doing ANYTHING that touches Notion workspaces, pages, or databases.
---

# Major Platform Resource: Notion

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__notion_search` — Search pages and databases in the Notion workspace. Args: `resourceId`, `query`, `filter?`, `sort?`, `startCursor?`, `pageSize?`
- `mcp__resources__notion_get_page` — Get a Notion page by ID. Args: `resourceId`, `pageId`
- `mcp__resources__notion_query_database` — Query a Notion database with optional filters and sorts. Args: `resourceId`, `databaseId`, `filter?`, `sorts?`, `startCursor?`, `pageSize?`
- `mcp__resources__notion_get_database` — Get a Notion database schema and properties. Args: `resourceId`, `databaseId`
- `mcp__resources__notion_get_block_children` — Get child blocks of a page or block. Args: `resourceId`, `blockId`, `startCursor?`, `pageSize?`
- `mcp__resources__notion_invoke` — Make any HTTP request to the Notion API. Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { notionClient } from "./clients";

// Generic invoke for any Notion API endpoint
const result = await notionClient.invoke("GET", "/v1/pages/PAGE_ID", "get-page");
if (result.ok && result.result.status === 200 && result.result.body.kind === "json") {
	const page = result.result.body.value;
}

// POST with body
const searchResult = await notionClient.invoke("POST", "/v1/search", "search-pages", {
	body: { type: "json", value: { query: "Meeting notes" } },
});

// Query a database with filters
const queryResult = await notionClient.invoke("POST", "/v1/databases/DB_ID/query", "query-tasks", {
	body: {
		type: "json",
		value: {
			filter: { property: "Status", select: { equals: "Done" } },
			sorts: [{ property: "Created", direction: "descending" }],
		},
	},
});
```

## Tips

- All Notion API requests automatically include the `Notion-Version: 2022-06-28` header
- Notion uses UUIDs for page/database/block IDs (with or without hyphens)
- Pagination uses `start_cursor` and `has_more` pattern
- Database queries use a filter object — refer to Notion API docs for filter syntax
- Rich text is returned as arrays of rich text objects, not plain strings
- **Rate limit**: ~3 requests/second per integration. Handle HTTP 429 with Retry-After header.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`

**Docs**: [Notion API Reference](https://developers.notion.com/reference)
