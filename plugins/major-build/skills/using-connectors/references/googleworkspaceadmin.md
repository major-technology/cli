# Major Platform Resource: Google Workspace Admin

Proxy-only connector for Google Workspace onboarding and offboarding: create, suspend, and delete users, manage groups and licenses, revoke sessions, transfer Drive and Calendar data, and change a departing user's Gmail settings. There are no native MCP tools and no generated TypeScript client. Everything goes through the generic HTTP proxy.

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use MCP tools or the HTTP proxy.

**Two ways to interact with Google Workspace Admin:**

1. **MCP tools** (direct, no code needed): `mcp__resources__http_proxy_get` and `mcp__resources__http_proxy_invoke`. Use `mcp__resources__list_resources` to find the resource ID.
2. **HTTP proxy** (Next.js apps): Use `createProxyFetch` from `@major-tech/resource-client/next`. See [using-http-proxy](http-proxy.md) for setup and usage.

**Framework note**: Next.js = `createProxyFetch` must be used in server-side code only (Server Components, Server Actions, Route Handlers).

---

## MCP Tools

- `mcp__resources__http_proxy_get` - Read-only GET. Runs without approval. Args: `resourceId`, `url`, `headers?`, `timeoutMs?`.
- `mcp__resources__http_proxy_invoke` - Every other method (POST, PUT, PATCH, DELETE). Requires user approval. Args: `resourceId`, `url`, `method`, `headers?`, `body?`, `timeoutMs?`.

This connector has no base URL. Always pass the **absolute** Google URL. Allowed hosts and path prefixes:

| Host | Path prefixes | Acts as |
| --- | --- | --- |
| `admin.googleapis.com` | `/admin/directory`, `/admin/datatransfer`, `/admin/reports` | The admin email set on the connector |
| `licensing.googleapis.com` | `/apps/licensing` | The admin email set on the connector |
| `gmail.googleapis.com` | `/gmail/v1/users/{email}/settings/...` | The user named in the path |

Any other host or path is rejected by the proxy.

## Rules

- **Writes need approval.** Every POST, PUT, PATCH, and DELETE goes through `http_proxy_invoke`, so the user approves each one. Do one logical step per call and tell the user what it does.
- **Gmail needs the user's email in the path. Never `me`.** The service account has no mailbox of its own, so `users/me` fails with `auth_injection_failed`. Use `users/jane@acme.com/settings/...`.
- **Gmail is settings only.** Only `/gmail/v1/users/{email}/settings/...` is allowed (auto-reply, forwarding, delegates). Other Gmail paths (messages, threads, labels, drafts) are rejected.
- **Suspend before delete.** Suspend, sign out, and revoke tokens first. Transfer data and set up forwarding or delegates before deleting. Deleting is hard to undo and removes the user's Gmail settings access.
- **Directory `userKey` accepts the primary email or the numeric user ID.** Data Transfer needs numeric IDs. Get the `id` field from `GET /admin/directory/v1/users/{userKey}`.
- **Bodies are JSON.** Use `body: { "type": "json", "value": { ... } }`.
- **Cannot do:** read or search mail, read Drive file contents, or run Google Vault exports. If the user needs a mail export, tell them to use Vault in the Admin console.
- **Fixed scopes.** The connector requests a fixed scope list (Directory user, user security, group, group member, org unit, mobile device action, Data Transfer, Reports audit read-only, Licensing, Gmail settings basic and sharing). Endpoints outside those scopes return 403.

## Onboarding

### Create a user

```
mcp__resources__http_proxy_invoke({
    resourceId: "<resource-id>",
    method: "POST",
    url: "https://admin.googleapis.com/admin/directory/v1/users",
    body: { type: "json", value: {
        primaryEmail: "jane.doe@acme.com",
        name: { givenName: "Jane", familyName: "Doe" },
        password: "<temporary password, 8-100 chars>",
        changePasswordAtNextLogin: true,
        orgUnitPath: "/Engineering"
    } },
    description: "Create Workspace user"
})
```

`primaryEmail`, `name.givenName`, `name.familyName`, and `password` are required. `orgUnitPath` is optional (defaults to `/`). The response includes the numeric `id`.

### Add to a group

```
POST https://admin.googleapis.com/admin/directory/v1/groups/{groupKey}/members
{ "email": "jane.doe@acme.com", "role": "MEMBER" }
```

`groupKey` is the group email, alias, or ID. `role` is `MEMBER`, `MANAGER`, or `OWNER`.

### Assign a license

```
POST https://licensing.googleapis.com/apps/licensing/v1/product/{productId}/sku/{skuId}/user
{ "userId": "jane.doe@acme.com" }
```

Example: `productId` `Google-Apps`, `skuId` `1010020027` (Business Starter), `1010020028` (Business Standard), `1010020025` (Business Plus). Other SKUs are listed in Google's licensing product and SKU IDs page.

## Offboarding

Recommended order: suspend, sign out, revoke tokens, remove from groups, transfer data, set Gmail auto-reply and forwarding or delegate, remove license (optional), delete, then add the old address as an alias on the manager.

### Suspend

```
PUT https://admin.googleapis.com/admin/directory/v1/users/{userKey}
{ "suspended": true }
```

### Sign out of all sessions

```
POST https://admin.googleapis.com/admin/directory/v1/users/{userKey}/signOut
```

No body.

### Revoke third-party tokens

List with `http_proxy_get`:

```
GET https://admin.googleapis.com/admin/directory/v1/users/{userKey}/tokens
```

Each item has `clientId`, `displayText`, and `scopes`. Then, for each item:

```
DELETE https://admin.googleapis.com/admin/directory/v1/users/{userKey}/tokens/{clientId}
```

### Remove from a group

List the user's groups with `GET https://admin.googleapis.com/admin/directory/v1/groups?userKey={userKey}`, then for each group:

```
DELETE https://admin.googleapis.com/admin/directory/v1/groups/{groupKey}/members/{memberKey}
```

`memberKey` is the member's primary email, alias, or ID.

### Transfer Drive and Calendar ownership (Data Transfer)

1. Get the application IDs (do not hardcode them; Google says its list can change):

```
GET https://admin.googleapis.com/admin/datatransfer/v1/applications
```

At the time of writing: Drive and Docs `55656082996`, Calendar `435070579839`.

2. Get the numeric `id` of the old user and the receiving user (usually the manager) with `GET https://admin.googleapis.com/admin/directory/v1/users/{userKey}`.

3. Start the transfer:

```
POST https://admin.googleapis.com/admin/datatransfer/v1/transfers
{
  "oldOwnerUserId": "<numeric id of departing user>",
  "newOwnerUserId": "<numeric id of manager>",
  "applicationDataTransfers": [
    {
      "applicationId": "55656082996",
      "applicationTransferParams": [
        { "key": "PRIVACY_LEVEL", "value": ["PRIVATE", "SHARED"] }
      ]
    },
    {
      "applicationId": "435070579839",
      "applicationTransferParams": [
        { "key": "RELEASE_RESOURCES", "value": ["TRUE"] }
      ]
    }
  ]
}
```

Drive `PRIVACY_LEVEL` takes `PRIVATE` and `SHARED`; send both to move all files (default is `SHARED` only). Calendar only moves future, non-private events with at least one guest or resource on the primary calendar. The transfer is asynchronous. Poll `GET https://admin.googleapis.com/admin/datatransfer/v1/transfers/{dataTransferId}` and check `overallTransferStatusCode`.

### Gmail auto-reply

Acts as the user in the path. Email required, never `me`.

```
PUT https://gmail.googleapis.com/gmail/v1/users/jane.doe@acme.com/settings/vacation
{
  "enableAutoReply": true,
  "responseSubject": "Jane Doe has left Acme",
  "responseBodyPlainText": "Jane no longer works at Acme. Please contact john@acme.com.",
  "restrictToContacts": false,
  "restrictToDomain": false
}
```

Optional `startTime` and `endTime` are epoch milliseconds as strings.

### Gmail forwarding

Two steps. First add the forwarding address, then enable auto-forwarding:

```
POST https://gmail.googleapis.com/gmail/v1/users/jane.doe@acme.com/settings/forwardingAddresses
{ "forwardingEmail": "manager@acme.com" }
```

```
PUT https://gmail.googleapis.com/gmail/v1/users/jane.doe@acme.com/settings/autoForwarding
{ "enabled": true, "emailAddress": "manager@acme.com", "disposition": "leaveInInbox" }
```

`disposition` is `leaveInInbox`, `archive`, `trash`, or `markRead`. Google requires a verified forwarding address when auto-forwarding is enabled. If the address is not accepted yet (`verificationStatus` is `pending`), the recipient must confirm it first. Read it back with `GET .../settings/forwardingAddresses/{email}` and wait for `accepted`. Addresses in the same Workspace domain are the safe choice.

### Gmail delegate

Gives the manager access to the departing user's mailbox.

```
POST https://gmail.googleapis.com/gmail/v1/users/jane.doe@acme.com/settings/delegates
{ "delegateEmail": "manager@acme.com" }
```

The delegate must be in the same Workspace organization and identified by primary address (not an alias). It can take up to a minute to become usable.

### Delete the user

```
DELETE https://admin.googleapis.com/admin/directory/v1/users/{userKey}
```

Do this last, after suspend and after data transfer finishes. Confirm with the user first.

### Add the old address as an alias on the manager

Only after the old user is deleted. Until then the address belongs to the old user and the call fails.

```
POST https://admin.googleapis.com/admin/directory/v1/users/{managerKey}/aliases
{ "alias": "jane.doe@acme.com" }
```

## TypeScript: `createProxyFetch`

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const gwsFetch = createProxyFetch({
    baseUrl: process.env.MAJOR_API_BASE_URL!,
    resourceId: process.env.GOOGLE_WORKSPACE_ADMIN_RESOURCE_ID!,
    majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

export async function suspendUser(userKey: string) {
    const res = await gwsFetch(
        `https://admin.googleapis.com/admin/directory/v1/users/${encodeURIComponent(userKey)}`,
        {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ suspended: true }),
        },
    );

    if (!res.ok) {
        throw new Error(`Suspend failed: ${res.status} ${await res.text()}`);
    }

    return res.json();
}
```

Use absolute URLs. Do not set `Authorization`; the proxy injects a token for the right user.

## Tips

- **Auth is automatic.** A service account with domain-wide delegation mints a token per request. Admin and Licensing calls act as the configured admin email. Gmail settings calls act as the user in the path.
- **Errors**: `auth_injection_failed` on a Gmail call usually means `me` or a non-email in the path, or a non-settings path. 403 from Google means the admin email is not an admin or the API is not enabled in the Cloud project. `unauthorized_client` means the client ID or scopes are not authorized in the Admin console.
- **Pagination**: Directory list endpoints use `pageToken` and `maxResults`. Use `customer=my_customer` for domain-wide lists.
- **Reference**: [Directory API](https://developers.google.com/workspace/admin/directory/reference/rest), [Data Transfer API](https://developers.google.com/workspace/admin/data-transfer/reference/rest), [License Manager API](https://developers.google.com/workspace/admin/licensing/reference/rest), [Gmail settings](https://developers.google.com/workspace/gmail/api/reference/rest/v1/users.settings).
