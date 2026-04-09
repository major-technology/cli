---
name: using-googlecalendar-connector
description: Implements Google Calendar event management and scheduling using generated clients and MCP tools. Use when doing ANYTHING that touches Google Calendar or gcal in any way, load this skill.
---

# Major Platform Resource: Google Calendar

## Setting Up a Google Calendar Connector

Google Calendar requires OAuth authentication before use.

### When the user asks you to set up Google Calendar or connect their calendar:

1. Call `mcp__resource-setup__request-resource-setup` with `subtype: "googlecalendar"` — this prompts the user to authenticate with Google
2. Once setup completes, the resource is ready to use

---

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "List all user accounts", "Check table schema"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-upcoming-events"`, never dynamic values like `` `${date}-events` ``.

---

## MCP Tools

- `mcp__resources__googlecalendar_list_calendars` — List all calendars accessible to the connected account. Args: `resourceId`
- `mcp__resources__googlecalendar_list_events` — List events with optional date filtering, search, and pagination. Args: `resourceId`, `calendarId?` (default: "primary"), `timeMin?`, `timeMax?`, `maxResults?`, `q?`, `singleEvents?`, `orderBy?`, `pageToken?`
- `mcp__resources__googlecalendar_create_event` — Create a new calendar event. Args: `resourceId`, `summary`, `startDateTime`, `endDateTime`, `calendarId?`, `location?`, `eventDescription?`, `timeZone?`, `attendees?`
- `mcp__resources__googlecalendar_invoke` — Make any HTTP request to the Google Calendar API v3 (for operations not covered by other tools). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { googleCalendarClient } from "./clients";

// invoke(method, path, invocationKey, options?)
// All paths are relative to https://www.googleapis.com/calendar/v3/

// List upcoming events
const result = await googleCalendarClient.invoke("GET", "calendars/primary/events", "list-events", {
	query: { timeMin: new Date().toISOString(), maxResults: "10", singleEvents: "true", orderBy: "startTime" },
});
if (result.ok && result.result.status === 200 && result.result.body.kind === "json") {
	const events = result.result.body.value.items;
}

// Create an event
const createResult = await googleCalendarClient.invoke("POST", "calendars/primary/events", "create-meeting", {
	body: {
		type: "json",
		value: {
			summary: "Team Standup",
			start: { dateTime: "2026-04-03T10:00:00-07:00" },
			end: { dateTime: "2026-04-03T10:30:00-07:00" },
			attendees: [{ email: "colleague@example.com" }],
		},
	},
});
```

## Tips

- **All paths are relative to `https://www.googleapis.com/calendar/v3/`** — e.g. use `calendars/primary/events`, not the full URL.
- **Use `singleEvents=true`** when listing events to expand recurring events into individual instances. This also enables `orderBy=startTime`.
- **Date format**: Use RFC3339 for datetime (`2026-04-02T10:00:00-07:00`) or `YYYY-MM-DD` for all-day events.
- **Default calendar**: Use `"primary"` as the calendar ID to target the user's main calendar.
- **Pagination**: Check `nextPageToken` in the response and pass it as `pageToken` to get the next page.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`
- **Common paths**: `calendars/primary/events` (list/create events), `users/me/calendarList` (list calendars), `calendars/{calendarId}/events/{eventId}` (get/update/delete event)
- **Scope presets**: The resource may be configured as "readonly" (can only read) or "readwrite" (can read and create/modify events). Write operations will fail with 403 if the resource is readonly.

**Docs**: [Google Calendar API Reference](https://developers.google.com/calendar/api/v3/reference)

---

## Per-User OAuth

Google Calendar uses per-user OAuth — each user must connect their own Google account (`requiresUserOAuth: true` in `list_resources`). Before using Google Calendar tools:

1. Call `setup-user-oauth` with the resource ID
2. If credentials are missing, present the user with the connect URL
3. **CLI users:** Tell them to run `major resource connect <resourceId> --environment <envId>`
4. Wait for the user to confirm they've connected, then retry the resource tools
