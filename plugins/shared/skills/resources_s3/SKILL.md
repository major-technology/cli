---
name: using-s3-connector
description: Implements Amazon S3 object operations, presigned URLs, and file uploads/downloads using generated clients and MCP tools. Use when doing ANYTHING that touches S3 or AWS S3 in any way, load this skill.
---

# Major Platform Resource: Amazon S3

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "List all user accounts", "Check table schema"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__s3_list_buckets` — List all accessible buckets. Args: `resourceId`
- `mcp__resources__s3_list_objects` — List objects with optional prefix/delimiter. Args: `resourceId`, `bucket`, `prefix?`, `delimiter?`, `maxKeys?`
- `mcp__resources__s3_get_object_metadata` — Get size, content type, last modified. Args: `resourceId`, `bucket`, `key`

## TypeScript Client

```typescript
import { storageClient } from "./clients";

// Generate presigned URL for upload
const uploadResult = await storageClient.invoke(
	{ command: "PutObject", key: "uploads/image.jpg", presignedUrl: true, expiresIn: 3600 },
	"generate-upload-url",
);
if (uploadResult.ok) {
	const url = uploadResult.result.presignedUrl;
	// Return URL to frontend for direct upload
}

// List objects
const listResult = await storageClient.invoke({ command: "ListObjectsV2", prefix: "uploads/" }, "list-uploads");
```

## Tips

- **Always use presigned URLs** for uploads and downloads — generate on server, return URL to frontend for direct S3 access
- **Never proxy file contents** through your application server
- S3 commands: `PutObject`, `GetObject`, `ListObjectsV2`, `DeleteObject`, `HeadObject`
- Compatible with AWS S3, MinIO, DigitalOcean Spaces, and other S3-compatible storage

**Docs**: [Amazon S3 Documentation](https://docs.aws.amazon.com/s3/)
