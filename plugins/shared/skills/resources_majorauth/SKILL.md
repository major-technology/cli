---
name: using-auth-connector
description: Share or revoke application access for users by email. Use whenever the app needs to manage user access to the app.
---

# Major Platform Resource: Major Auth Connector

## Common: Interacting with Resources

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "Share access with user", "Revoke user access"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"share-user-access"`, never dynamic values like `` `${date}-share` ``.

---

## How to Use

The Major Auth Connector is a **managed resource** that exists by default in every organization. To use it:

1. Call `mcp__resources__list_resources` to discover available resources — look for the one with subtype `majorauth` (named "Major Auth Connector").
2. Call `mcp__resource-tools__add-resource-client` with that `resourceId` to generate a typed `MajorAuthResourceClient`.
3. Use the generated client in your app code to share or revoke access.

## MCP Tools

- `mcp__resources__majorauth_share_access` — Grant a user view access to the current app by email. Creates user account and org membership if needed. Args: `resourceId`, `email`
- `mcp__resources__majorauth_revoke_access` — Revoke a user's view access to the current app by email. Only removes app-level access; does not affect org membership. Args: `resourceId`, `email`

## TypeScript Client

```typescript
import { authClient } from "./clients";

// Grant a user view access to the app by email
// Creates their account and org membership if they don't exist
const shareResult = await authClient.shareAccess("user@example.com", "share-user-access");
if (shareResult.ok) {
	console.log("Access granted:", shareResult.result.success);
}

// Revoke a user's app access (does NOT remove them from the org)
const revokeResult = await authClient.revokeAccess("user@example.com", "revoke-user-access");
if (revokeResult.ok) {
	console.log("Access revoked:", revokeResult.result.success);
}
```

## Tips

- **View access only**: `shareAccess` grants `Application:User` role which gives view access to the app. It does not grant edit or admin permissions.
- **Auto-creates users**: If the email doesn't have an account, one is automatically created and added to the organization.
- **Idempotent**: Calling `shareAccess` for a user who already has access is safe and will not error.
- **Revoke is app-scoped**: `revokeAccess` only removes the user's access to this specific app. It does not remove them from the organization or affect any other apps.
- **If a user is NOT invited through this connector, they will get redirected out of the app when they try to enter the app**
