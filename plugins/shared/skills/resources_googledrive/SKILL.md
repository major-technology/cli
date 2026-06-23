---
name: using-googledrive-connector
description: Implements Google Drive file listing, reading, and management using generated clients and MCP tools. Use when doing ANYTHING that touches Google Drive, files, or documents in any way, load this skill.
---

# Major Platform Resource: Google Drive

## Setting Up a Google Drive Connector

Google Drive requires OAuth authentication before use.

### When the user asks you to set up Google Drive or connect their files:

1. Call `mcp__resource-setup__request-resource-setup` with `subtype: "googledrive"` — this prompts the user to authenticate with Google
2. Once setup completes, the resource is ready to use

---

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "List project documents", "Get file contents"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-project-files"`, never dynamic values like `` `${folder}-files` ``.

---

## MCP Tools

- `mcp__resources__googledrive_list_files` — Search and list files. Args: `resourceId`, `query?` (Drive search syntax), `maxResults?`, `pageToken?`
- `mcp__resources__googledrive_get_file` — Get file metadata by ID. Args: `resourceId`, `fileId`
- `mcp__resources__googledrive_get_file_content` — Export a Google Docs/Sheets/Slides file to a specified format. Args: `resourceId`, `fileId`, `mimeType?` (default: "text/plain"). For binary files, use `googledrive_invoke` with `alt=media`.
- `mcp__resources__googledrive_list_shared_drives` — List shared drives. Args: `resourceId`
- `mcp__resources__googledrive_invoke` — Make any HTTP request to the Google Drive API v3 (for operations not covered by other tools). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { googleDriveClient } from "./clients";

// invoke(method, path, invocationKey, options?)
// All paths are relative to https://www.googleapis.com/drive/v3/

// List recent files
const result = await googleDriveClient.invoke("GET", "files", "list-files", {
	query: { pageSize: "10", fields: "files(id,name,mimeType,modifiedTime)", orderBy: "modifiedTime desc" },
});
if (result.ok && result.result.status === 200 && result.result.body.kind === "json") {
	const files = result.result.body.value.files;
}

// Search for spreadsheets
const searchResult = await googleDriveClient.invoke("GET", "files", "search-spreadsheets", {
	query: { q: "mimeType='application/vnd.google-apps.spreadsheet'", fields: "files(id,name)" },
});

// Export a Google Doc as plain text
const exportResult = await googleDriveClient.invoke("GET", "files/FILE_ID/export", "export-doc", {
	query: { mimeType: "text/plain" },
});
```

## Tips

- **All paths are relative to `https://www.googleapis.com/drive/v3/`** — e.g. use `files`, not the full URL.
- **Drive search syntax**: `name contains 'report'`, `mimeType = 'application/vnd.google-apps.spreadsheet'`, `modifiedTime > '2024-01-01'`, `'FOLDER_ID' in parents`, `trashed = false`. Combine with `and`/`or`.
- **Google Workspace MIME types**: `application/vnd.google-apps.document` (Docs), `application/vnd.google-apps.spreadsheet` (Sheets), `application/vnd.google-apps.presentation` (Slides), `application/vnd.google-apps.folder` (Folder)
- **Export MIME types** (for `get_file_content`): `text/plain`, `text/csv`, `application/pdf`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document` (.docx), `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` (.xlsx)
- **Binary file download**: Use `googledrive_invoke` with path `files/{id}?alt=media` to download non-Google files directly.
- **Pagination**: Check `nextPageToken` in the response and pass it as `pageToken` to get the next page.
- **Fields parameter**: Use `fields` query param to limit response size, e.g. `fields=files(id,name,mimeType,modifiedTime,size)`.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`
- **Scope presets**: "readonly" (view files) or "readwrite" (view all files + manage app-created files). Write operations fail with 403 on readonly.

**Docs**: [Google Drive API Reference](https://developers.google.com/drive/api/reference/rest/v3)
