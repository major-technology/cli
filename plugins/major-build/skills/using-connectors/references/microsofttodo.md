# Major Platform Resource: Microsoft To Do

Microsoft To Do is a proxy-only resource backed by Microsoft Graph — every call goes through Major's HTTP proxy. There is **no typed `microsofttodo_*` MCP tool and no generated TypeScript client**. Use a **leading-slash relative path** under `https://graph.microsoft.com`, for example `/v1.0/me/todo/lists`.

**Security**: Never set the `Authorization` header — the proxy injects it. Reserved headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`, `X-Major-*`) are stripped on the way out.

---

## Setting Up a Microsoft To Do Connector

Microsoft To Do requires Microsoft OAuth before use.

1. Call `mcp__plugin_major_major__request_resource_setup` with `connectorId: "microsofttodo"` — this prompts the user to sign in with Microsoft.
2. Once setup completes, the resource is ready to use.

The connection has a scope preset. `readonly` (`Tasks.Read`) can read task lists and tasks; `readwrite` (`Tasks.ReadWrite`) can also create, update, and delete them. A `readonly` connection returns `403` on writes. A Microsoft 365 admin may need to approve task access for the organization.

---

## MCP Tools

Use the generic HTTP proxy tools — there are no Microsoft To Do-specific MCP tools.

- `mcp__resources__http_proxy_get` — Read-only GET. Args: `resourceId`, `url` (leading-slash path), `headers?`, `timeoutMs?`
- `mcp__resources__http_proxy_invoke` — Any HTTP method. Args: `resourceId`, `method`, `url` (leading-slash path), `headers?`, `body?`, `timeoutMs?`

See [using-http-proxy](http-proxy.md) for the full proxy reference.

## HTTP Proxy via `createProxyFetch` (Next.js)

For app code, use the generic `createProxyFetch` as a plain `fetch` wrapper. Pass **leading-slash paths** instead of a full URL.

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.MICROSOFT_TODO_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

// Task lists
const listsRes = await proxyFetch("/v1.0/me/todo/lists");
if (!listsRes.ok) throw new Error(`Microsoft To Do ${listsRes.status}: ${await listsRes.text()}`);
const { value: lists } = await listsRes.json();

// Open tasks in the default "Tasks" list
const defaultList = lists.find((l: { wellknownListName: string }) => l.wellknownListName === "defaultList");
const tasksRes = await proxyFetch(`/v1.0/me/todo/lists/${defaultList.id}/tasks`);
const { value: tasks } = await tasksRes.json();
const openTasks = tasks.filter((t: { status: string }) => t.status !== "completed");

// Create a task with a due date
const created = await proxyFetch(`/v1.0/me/todo/lists/${defaultList.id}/tasks`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    title: "Send the Q3 report",
    importance: "high",
    dueDateTime: { dateTime: "2026-01-05T17:00:00", timeZone: "America/Los_Angeles" },
    body: { content: "Attach the final numbers.", contentType: "text" },
  }),
});
```

**Rules:**

- `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they're set, don't ask the user to provide them. `MICROSOFT_TODO_RESOURCE_ID` is the UUID of the connected Microsoft To Do resource.
- **Always pass a leading-slash path.** Do not hardcode `https://graph.microsoft.com/...`.
- **`resourceId` must be a static string literal** — the query extractor only tracks proxy calls when the resource id is known at build time.
- Server-side only (Server Components, Route Handlers, Server Actions). The `/next` subpath uses `next/headers`.
- Do NOT set the `Authorization` header — the proxy injects upstream auth.

## Common Microsoft To Do endpoints

| What                      | Method | Path                                                                  |
| ------------------------- | ------ | --------------------------------------------------------------------- |
| List task lists           | GET    | `/v1.0/me/todo/lists`                                                 |
| Create a task list        | POST   | `/v1.0/me/todo/lists`                                                 |
| Rename a task list        | PATCH  | `/v1.0/me/todo/lists/{listId}`                                        |
| Delete a task list        | DELETE | `/v1.0/me/todo/lists/{listId}`                                        |
| List tasks                | GET    | `/v1.0/me/todo/lists/{listId}/tasks`                                  |
| Get a task                | GET    | `/v1.0/me/todo/lists/{listId}/tasks/{taskId}`                         |
| Create a task             | POST   | `/v1.0/me/todo/lists/{listId}/tasks`                                  |
| Update / complete a task  | PATCH  | `/v1.0/me/todo/lists/{listId}/tasks/{taskId}`                         |
| Delete a task             | DELETE | `/v1.0/me/todo/lists/{listId}/tasks/{taskId}`                         |
| Checklist items (subtasks) | GET/POST | `/v1.0/me/todo/lists/{listId}/tasks/{taskId}/checklistItems`        |
| Linked resources          | GET/POST | `/v1.0/me/todo/lists/{listId}/tasks/{taskId}/linkedResources`       |
| Changed tasks (delta)     | GET    | `/v1.0/me/todo/lists/{listId}/tasks/delta`                            |

A task list takes `{ displayName }`. A task takes `title`, `body { content, contentType }`, `importance` (`low` / `normal` / `high`), `status` (`notStarted` / `inProgress` / `completed` / `waitingOnOthers` / `deferred`), `dueDateTime`, `reminderDateTime`, `isReminderOn`, and `categories`. A checklist item takes `{ displayName, isChecked }`.

## Tips

- **Complete a task with PATCH**: send `{ "status": "completed" }`. Send `{ "status": "notStarted" }` to reopen it.
- **Built-in lists**: `wellknownListName` is `defaultList` for the built-in "Tasks" list, `flaggedEmails` for tasks from flagged Outlook emails, and `none` for lists the user created. The built-in lists cannot be renamed or deleted.
- **Ownership**: each list has `isOwner` (the user owns the list) and `isShared` (the list is shared with other users).
- **Dates are `dateTimeTimeZone` objects** (`{ dateTime, timeZone }`), not plain ISO strings.
- **Filter tasks in code.** Microsoft documents only that list endpoints support "some" OData query parameters; it does not list which `$filter` / `$orderby` expressions work on tasks. Fetch the list and filter on `status`, `importance`, or `dueDateTime` in code. If you try a server-side `$filter`, check `res.ok` and fall back to filtering in code.
- **Delta** (`/tasks/delta`) supports `$select`, `$top`, and `$expand`. Its only `$filter` / `$orderby` expressions are on `receivedDateTime`. Save the `@odata.deltaLink` and call it next time to get only the changes.
- **A task id can change** when the task moves to another list.
- **`readonly` connections return `403` on writes.** Ask the user to reconnect with the `readwrite` preset.
- **Pagination** uses `@odata.nextLink` in the response. Request the next page by passing its path and query as the next `url` (strip `https://graph.microsoft.com`).
- **Non-2xx upstream statuses are passed through** — always inspect `res.ok` / `res.status` before treating the body as success.

**Docs**: [Microsoft Graph To Do API](https://learn.microsoft.com/en-us/graph/api/resources/todo-overview)
