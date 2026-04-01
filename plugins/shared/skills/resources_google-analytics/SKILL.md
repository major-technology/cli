---
name: using-google-analytics-connector
description: Implements Google Analytics (GA4) reporting, metadata exploration, and account management using generated clients and MCP tools. Use when doing ANYTHING that touches Google Analytics or GA4 in any way, load this skill.
---

# Major Platform Resource: Google Analytics

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "List all user accounts", "Check table schema"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-page-views"`, never dynamic values like `` `${date}-report` ``.

---

## MCP Tools

- `mcp__resources__googleanalytics_run_report` — Run a GA4 report with dimensions, metrics, and date ranges. Args: `resourceId`, `dimensions`, `metrics`, `dateRanges`, `orderBys?`, `limit?`, `offset?`
- `mcp__resources__googleanalytics_get_metadata` — Get available dimensions and metrics for the property. Args: `resourceId`
- `mcp__resources__googleanalytics_run_realtime_report` — Run a realtime report showing live activity. Args: `resourceId`, `dimensions?`, `metrics`, `limit?`
- `mcp__resources__googleanalytics_list_accounts` — List all GA4 accounts. Args: `resourceId`, `pageSize?`, `pageToken?`
- `mcp__resources__googleanalytics_list_properties` — List GA4 properties, optionally by account. Args: `resourceId`, `accountId?`, `pageSize?`, `pageToken?`
- `mcp__resources__googleanalytics_list_data_streams` — List data streams for a property. Args: `resourceId`, `propertyId?`, `pageSize?`, `pageToken?`
- `mcp__resources__googleanalytics_invoke` — Execute any GA4 operation. Args: `resourceId`, `operation`, + operation-specific args

## TypeScript Client

```typescript
import { gaClient } from "./clients";

// runReport(dimensions, metrics, dateRanges, invocationKey, options?)
const result = await gaClient.runReport(
	[{ name: "country" }, { name: "city" }],
	[{ name: "activeUsers" }, { name: "sessions" }],
	[{ startDate: "30daysAgo", endDate: "today" }],
	"traffic-by-location",
	{ limit: 100 },
);

// getMetadata(invocationKey)
const metadata = await gaClient.getMetadata("discover-dimensions");

// listAccounts(invocationKey, options?)
const accounts = await gaClient.listAccounts("list-ga-accounts");

// listProperties(invocationKey, accountId?, options?)
const properties = await gaClient.listProperties("list-ga-properties", "accounts/12345");

// runRealtimeReport(metrics, invocationKey, dimensions?, limit?)
const realtime = await gaClient.runRealtimeReport([{ name: "activeUsers" }], "live-users");
```

## Tips

- **Property ID format**: Use the numeric GA4 property ID (e.g., `123456789`), not the measurement ID (G-XXXXXXXX). Find it in GA4 Admin > Property Settings.
- **Date ranges**: Use relative dates like `today`, `yesterday`, `7daysAgo`, `30daysAgo`, or absolute dates in `YYYY-MM-DD` format.
- **Discover available data**: Use `get_metadata` first to see which dimensions and metrics are available for the property before running reports.
- **Dimension/metric names**: Use API names like `country`, `city`, `activeUsers`, `sessions`, `screenPageViews` — not display names.
- **Realtime reports**: Do not support date ranges (they show live data only). Only a subset of dimensions/metrics are available.
- **Pagination**: Use `limit` and `offset` for report results, `pageSize` and `pageToken` for list operations.

**Docs**: [GA4 Data API](https://developers.google.com/analytics/devguides/reporting/data/v1) | [GA4 Admin API](https://developers.google.com/analytics/devguides/config/admin/v1)
