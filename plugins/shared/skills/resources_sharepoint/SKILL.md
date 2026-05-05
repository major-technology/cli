---
name: using-sharepoint-connector
description: Implements Microsoft SharePoint access — sites, lists, document libraries, and file operations — using generated clients and MCP tools. Use when doing ANYTHING that touches SharePoint, OneDrive for Business, or Microsoft Graph Sites/Files API.
---

# Major Platform Resource: SharePoint

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** Always read the actual client source code to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only. Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.json`.

**Invocation keys must be static strings** — use descriptive literals like `"list-team-sites"`, never dynamic values.

---

## MCP Tools

- `mcp__resources__sharepoint_list_sites` — List SharePoint sites accessible to the connected account. Args: `resourceId`, `search?`, `options?`
- `mcp__resources__sharepoint_get_site_items` — Get items from a SharePoint list. Args: `resourceId`, `siteId`, `listId`, `options?`
- `mcp__resources__sharepoint_search_drive_items` — Search for files across SharePoint drives. Args: `resourceId`, `query`, `options?`
- `mcp__resources__sharepoint_get` — Generic GET request to any Microsoft Graph endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__sharepoint_invoke` — Generic HTTP request to Microsoft Graph (for write operations). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { spClient } from "./clients";

// List sites
const sites = await spClient.invoke("GET", "/v1.0/sites?search=*", "list-all-sites");

// Get items from a SharePoint list
const items = await spClient.invoke(
  "GET",
  "/v1.0/sites/{site-id}/lists/{list-id}/items?$expand=fields",
  "get-list-items"
);

// Search for files
const files = await spClient.invoke(
  "GET",
  "/v1.0/sites/{site-id}/drive/root/search(q='quarterly report')",
  "search-files"
);

// Create a list item (body is just a plain object — no wrapper needed)
const newItem = await spClient.invoke(
  "POST",
  "/v1.0/sites/{site-id}/lists/{list-id}/items",
  "create-item",
  { body: { fields: { Title: "New Item", Status: "Active" } } }
);
```

## Tips

- **Microsoft Graph API**: All paths are relative to `https://graph.microsoft.com`. Use `/v1.0/` prefix for stable endpoints.
- **Admin consent**: These scopes use delegated permissions and do NOT require admin consent by default. However, some Microsoft 365 tenants disable user consent org-wide — in that case, a tenant admin will need to approve the app once.
- **Common SharePoint paths**:
  - Sites: `/v1.0/sites?search=keyword`, `/v1.0/sites/{hostname}:/{server-relative-path}`
  - Lists: `/v1.0/sites/{site-id}/lists`, `/v1.0/sites/{site-id}/lists/{list-id}/items`
  - Drives: `/v1.0/sites/{site-id}/drives`, `/v1.0/sites/{site-id}/drive/root/children`
  - Files: `/v1.0/sites/{site-id}/drive/items/{item-id}`, `/v1.0/sites/{site-id}/drive/root:/{path}`
- **OData queries**: Use `$select`, `$filter`, `$expand`, `$top`, `$orderby` as query parameters
- **Pagination**: Graph API uses `@odata.nextLink` for pagination — pass the full URL to `invoke` for subsequent pages

**Docs**: [Microsoft Graph SharePoint API Reference](https://learn.microsoft.com/en-us/graph/api/resources/sharepoint?view=graph-rest-1.0)
