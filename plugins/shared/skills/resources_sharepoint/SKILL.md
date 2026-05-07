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

- `mcp__resources__sharepoint_list_sites` — List SharePoint sites accessible to the connected account. Args: `resourceId`, `search?`, `top?`, `select?`
- `mcp__resources__sharepoint_get_site_items` — Get items from a SharePoint list. Args: `resourceId`, `siteId`, `listId`, `select?`, `filter?`, `expand?`, `top?`
- `mcp__resources__sharepoint_search_drive_items` — Search for files across SharePoint drives. Args: `resourceId`, `query`, `top?`
- `mcp__resources__sharepoint_get_file_download_url` — Get a pre-authenticated download URL for a file. Args: `resourceId`, `siteId`, `itemId`
- `mcp__resources__sharepoint_create_upload_session` — Create a pre-authenticated upload session. Args: `resourceId`, `siteId`, `fileName`, `parentPath?`, `conflictBehavior?`
- `mcp__resources__sharepoint_get` — Generic GET request to any Microsoft Graph endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__sharepoint_invoke` — Generic HTTP request to Microsoft Graph (for JSON write operations). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

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

## File Downloads and Uploads

SharePoint files must be downloaded and uploaded using **pre-authenticated URLs** (presigned URLs), not through the connector's invoke method. The connector brokers the authenticated handshake with Microsoft Graph to obtain these URLs, but the actual binary transfer goes directly between your app and Microsoft's servers.

### Downloading a file

Use `get_file_download_url` (MCP) or request the `@microsoft.graph.downloadUrl` property (client) to get a short-lived presigned URL, then fetch the file directly from that URL.

**Via MCP tool:**
```
mcp__resources__sharepoint_get_file_download_url({ resourceId, siteId, itemId })
→ { id, name, size, "@microsoft.graph.downloadUrl": "https://..." }
```

**Via TypeScript client:**
```typescript
import { spClient } from "./clients";

// Step 1: Get the presigned download URL through the connector
const result = await spClient.invoke(
  "GET",
  "/v1.0/sites/{site-id}/drive/items/{item-id}?select=id,name,size,@microsoft.graph.downloadUrl",
  "get-download-url"
);
if (!result.ok) throw new Error(result.error.message);

const downloadUrl = result.json["@microsoft.graph.downloadUrl"];

// Step 2: Download the file directly — no auth headers needed
const fileResponse = await fetch(downloadUrl);
const fileBuffer = await fileResponse.arrayBuffer();
```

### Uploading a file

Use `create_upload_session` (MCP) or POST to `createUploadSession` (client) to get a presigned upload URL, then PUT file bytes directly to that URL.

**Via MCP tool:**
```
mcp__resources__sharepoint_create_upload_session({ resourceId, siteId, fileName: "report.pdf", parentPath: "Documents/Reports" })
→ { uploadUrl: "https://...", expirationDateTime: "..." }
```

**Via TypeScript client:**
```typescript
import { spClient } from "./clients";
import fs from "fs";

// Step 1: Create an upload session through the connector
const session = await spClient.invoke(
  "POST",
  "/v1.0/sites/{site-id}/drive/root:/Documents/report.pdf:/createUploadSession",
  "create-upload-session",
  {
    body: {
      item: {
        "@microsoft.graph.conflictBehavior": "rename",
        name: "report.pdf",
      },
    },
  }
);
if (!session.ok) throw new Error(session.error.message);

const uploadUrl = session.json.uploadUrl;

// Step 2: PUT the file bytes directly — no auth headers needed
const fileBuffer = fs.readFileSync("./report.pdf");
const uploadResponse = await fetch(uploadUrl, {
  method: "PUT",
  headers: {
    "Content-Length": String(fileBuffer.byteLength),
    "Content-Range": `bytes 0-${fileBuffer.byteLength - 1}/${fileBuffer.byteLength}`,
  },
  body: fileBuffer,
});
const uploaded = await uploadResponse.json(); // returns the driveItem
```

For files larger than 4MB, split into ~10MB chunks and PUT each with the appropriate `Content-Range` header. The upload session URL handles ordering and resumability automatically.

## Tips

- **Microsoft Graph API**: All paths are relative to `https://graph.microsoft.com`. Use `/v1.0/` prefix for stable endpoints.
- **File operations**: Always use the presigned URL tools (`get_file_download_url`, `create_upload_session`) for binary file transfer. The generic `invoke` tool only handles JSON request/response bodies.
- **Admin consent**: These scopes use delegated permissions and do NOT require admin consent by default. However, some Microsoft 365 tenants disable user consent org-wide — in that case, a tenant admin will need to approve the app once.
- **Common SharePoint paths**:
  - Sites: `/v1.0/sites?search=keyword`, `/v1.0/sites/{hostname}:/{server-relative-path}`
  - Lists: `/v1.0/sites/{site-id}/lists`, `/v1.0/sites/{site-id}/lists/{list-id}/items`
  - Drives: `/v1.0/sites/{site-id}/drives`, `/v1.0/sites/{site-id}/drive/root/children`
  - Files: `/v1.0/sites/{site-id}/drive/items/{item-id}`, `/v1.0/sites/{site-id}/drive/root:/{path}`
- **OData queries**: Use `$select`, `$filter`, `$expand`, `$top`, `$orderby` as query parameters
- **Pagination**: Graph API uses `@odata.nextLink` for pagination — pass the full URL to `invoke` for subsequent pages

**Docs**: [Microsoft Graph SharePoint API Reference](https://learn.microsoft.com/en-us/graph/api/resources/sharepoint?view=graph-rest-1.0)
