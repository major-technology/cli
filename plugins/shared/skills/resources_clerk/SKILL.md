---
name: using-clerk-connector
description: Implements Clerk Backend API requests with automatic Bearer Token auth using generated clients and MCP tools. Use when doing ANYTHING that touches a Clerk resource in any way, load this skill.
---

# Major Platform Resource: Clerk Backend API

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Three ways to interact with Clerk:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).
3. **HTTP proxy** (Next.js apps): Use `createProxyFetch` from `@major-tech/resource-client/next` to call the Clerk API directly with automatic auth injection. See [using-http-proxy](../http-proxy/SKILL.md) for setup and usage — preferred when you need to hit endpoints not covered by MCP tools or the typed client, or when using an official SDK that accepts a custom `fetch`.

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-clerk-users"`, never dynamic values.

---

## MCP Tools

- `mcp__resources__clerk_get` — Make a GET request to any Clerk Backend API endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__clerk_list_users` — List users with optional filtering. Args: `resourceId`, `emailAddress?`, `limit?`, `offset?`
- `mcp__resources__clerk_get_user` — Get a single user by ID. Args: `resourceId`, `userId`
- `mcp__resources__clerk_list_organizations` — List organizations. Args: `resourceId`, `limit?`, `offset?`
- `mcp__resources__clerk_invoke` — Make any HTTP request to the Clerk API (including writes). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { clerkClient } from "./clients";

// invoke(method, path, invocationKey, options?)
const result = await clerkClient.invoke("GET", "/v1/users", "list-users", {
    query: { limit: "10", offset: "0" },
});
if (result.ok) {
    const response = result.result;
    // response: { kind: "api", status: number, body: { kind: "json"|"text"|"binary", value: ... } }
}

// POST example - create an invitation
await clerkClient.invoke("POST", "/v1/invitations", "create-invitation", {
    body: { type: "json", value: { email_address: "user@example.com" } },
});
```

## Example: Search for Users by Email

**Using MCP tools (no code needed):**

Call `mcp__resources__clerk_list_users` with the `emailAddress` filter:

```
mcp__resources__clerk_list_users({
    resourceId: "<your-clerk-resource-id>",
    emailAddress: "jane@example.com",
    limit: "10"
})
```

**Using the TypeScript client in a Next.js Server Action:**

```typescript
"use server";

import { clerkClient } from "./clients";

interface ClerkUser {
    id: string;
    first_name: string | null;
    last_name: string | null;
    email_addresses: { email_address: string }[];
    created_at: number;
}

export async function searchUsersByEmail(email: string) {
    const result = await clerkClient.invoke("GET", "/v1/users", "search-users-by-email", {
        query: { email_address: email, limit: "20" },
    });

    if (!result.ok) {
        throw new Error(result.error.message);
    }

    const users = result.result.body.value as ClerkUser[];

    return users.map((u) => ({
        id: u.id,
        name: [u.first_name, u.last_name].filter(Boolean).join(" "),
        email: u.email_addresses[0]?.email_address ?? "",
        createdAt: new Date(u.created_at),
    }));
}
```

## Tips

- **Base URL is always `https://api.clerk.com`** — paths should include the version prefix (e.g. `/v1/users`)
- **Auth is automatic** — the Secret Key is sent as a Bearer token on every request
- **Clerk API version**: The connector targets the Clerk Backend API. Refer to [Clerk Backend API docs](https://clerk.com/docs/reference/backend-api) for endpoint details
- **Common endpoints**: `/v1/users`, `/v1/organizations`, `/v1/invitations`, `/v1/sessions`, `/v1/clients`
- **Pagination**: Most list endpoints support `limit` and `offset` query parameters
