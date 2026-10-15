# Major Platform Resource: Outlook Calendar

Outlook Calendar is a proxy-only resource backed by Microsoft Graph — every call goes through Major's HTTP proxy. There is **no typed `outlookcalendar_*` MCP tool and no generated TypeScript client**. Use a **leading-slash relative path** under `https://graph.microsoft.com`, for example `/v1.0/me/calendarView?startDateTime=2026-01-01T00:00:00Z&endDateTime=2026-01-08T00:00:00Z`.

**Security**: Never set the `Authorization` header — the proxy injects it. Reserved headers (`Authorization`, `Cookie`, `Host`, `Forwarded`, `X-Forwarded-*`, `X-Real-Ip`, `X-Major-*`) are stripped on the way out.

---

## Setting Up an Outlook Calendar Connector

Outlook Calendar requires Microsoft OAuth before use.

1. Call `mcp__plugin_major_major__request_resource_setup` with `connectorId: "outlookcalendar"` — this prompts the user to sign in with Microsoft.
2. Once setup completes, the resource is ready to use.

The connection has a scope preset. `readonly` can read calendars and events; `readwrite` can also create, update, and delete events. A `readonly` connection returns `403` on writes. A Microsoft 365 admin may need to approve calendar access for the organization.

---

## MCP Tools

Use the generic HTTP proxy tools — there are no Outlook Calendar-specific MCP tools.

- `mcp__resources__http_proxy_get` — Read-only GET. Args: `resourceId`, `url` (leading-slash path), `headers?`, `timeoutMs?`
- `mcp__resources__http_proxy_invoke` — Any HTTP method. Args: `resourceId`, `method`, `url` (leading-slash path), `headers?`, `body?`, `timeoutMs?`

See [using-http-proxy](http-proxy.md) for the full proxy reference.

## HTTP Proxy via `createProxyFetch` (Next.js)

For app code, use the generic `createProxyFetch` as a plain `fetch` wrapper. Pass **leading-slash paths** instead of a full URL.

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.OUTLOOK_CALENDAR_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

// Events in a time window, with recurring events expanded
const res = await proxyFetch(
  "/v1.0/me/calendarView?startDateTime=2026-01-01T00:00:00Z&endDateTime=2026-01-08T00:00:00Z&$orderby=start/dateTime",
  { headers: { Prefer: 'outlook.timezone="America/Los_Angeles"' } },
);
if (!res.ok) throw new Error(`Outlook Calendar ${res.status}: ${await res.text()}`);
const { value: events } = await res.json();

// Create an event with a Teams link
const created = await proxyFetch("/v1.0/me/events", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    subject: "Planning",
    start: { dateTime: "2026-01-05T10:00:00", timeZone: "America/Los_Angeles" },
    end: { dateTime: "2026-01-05T10:30:00", timeZone: "America/Los_Angeles" },
    attendees: [{ emailAddress: { address: "teammate@example.com" }, type: "required" }],
    isOnlineMeeting: true,
  }),
});
```

**Rules:**

- `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they're set, don't ask the user to provide them. `OUTLOOK_CALENDAR_RESOURCE_ID` is the UUID of the connected Outlook Calendar resource.
- **Always pass a leading-slash path.** Do not hardcode `https://graph.microsoft.com/...`.
- **`resourceId` must be a static string literal** — the query extractor only tracks proxy calls when the resource id is known at build time.
- Server-side only (Server Components, Route Handlers, Server Actions). The `/next` subpath uses `next/headers`.
- Do NOT set the `Authorization` header — the proxy injects upstream auth.

## Common Outlook Calendar endpoints

| What                      | Method | Path                                                                          |
| ------------------------- | ------ | ----------------------------------------------------------------------------- |
| List calendars            | GET    | `/v1.0/me/calendars`                                                          |
| Events in a time window   | GET    | `/v1.0/me/calendarView?startDateTime={iso}&endDateTime={iso}`                 |
| Same, on one calendar     | GET    | `/v1.0/me/calendars/{calendarId}/calendarView?startDateTime={iso}&endDateTime={iso}` |
| Get an event              | GET    | `/v1.0/me/events/{eventId}`                                                   |
| Create an event           | POST   | `/v1.0/me/events` (or `/v1.0/me/calendars/{calendarId}/events`)               |
| Update an event           | PATCH  | `/v1.0/me/events/{eventId}`                                                   |
| Delete an event           | DELETE | `/v1.0/me/events/{eventId}`                                                   |
| Free/busy                 | POST   | `/v1.0/me/calendar/getSchedule`                                               |
| Accept a meeting          | POST   | `/v1.0/me/events/{eventId}/accept`                                            |
| Decline a meeting         | POST   | `/v1.0/me/events/{eventId}/decline`                                           |

`getSchedule` takes `{ schedules: ["a@example.com"], startTime: { dateTime, timeZone }, endTime: { dateTime, timeZone }, availabilityViewInterval: 30 }`. Accept and decline take `{ comment?, sendResponse? }`.

## Tips

- **`calendarView` expands recurring events; `/events` does not.** Use `calendarView` to list what happens in a window. It requires `startDateTime` and `endDateTime`.
- **Set the time zone with a header.** Send `Prefer: outlook.timezone="America/Los_Angeles"` to get event times in that zone. Without it, times come back in UTC.
- **Online meetings**: set `isOnlineMeeting: true` on create to get a Teams link in `onlineMeeting.joinUrl`.
- **`readonly` connections return `403` on writes.** Ask the user to reconnect with the `readwrite` preset.
- **Pagination** uses `@odata.nextLink` in the response. Request the next page by passing its path and query as the next `url` (strip `https://graph.microsoft.com`). Use `$top` to set the page size.
- **Trim responses** with `$select` (for example `$select=subject,start,end,organizer`).
- **Non-2xx upstream statuses are passed through** — always inspect `res.ok` / `res.status` before treating the body as success.

**Docs**: [Microsoft Graph calendar API](https://learn.microsoft.com/en-us/graph/api/resources/calendar)
