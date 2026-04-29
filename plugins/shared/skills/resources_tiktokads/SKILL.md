---
name: using-tiktokads-connector
description: Implements TikTok Marketing API access for ad accounts, campaigns, and ad reports using generated clients and MCP tools. Use when doing ANYTHING that touches the TikTok Marketing / Ads Manager API — advertisers, campaigns, ad groups, ads, or reporting — in any way, load this skill.
---

# Major Platform Resource: TikTok Marketing API

Reference: https://business-api.tiktok.com/portal/docs

## Common: Interacting with Resources

**Security**: Never connect directly to TikTok APIs with raw credentials. Never put OAuth tokens in code, logs, env vars, prompts, or user-visible output. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__tiktokads_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and exact signatures before writing app code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** - use descriptive literals like `"fetch-tiktok-campaigns"`, never dynamic values like `` `${advertiserId}-campaigns` ``.

---

## Connector Model

- The connector is OAuth-token scoped, not bound to one advertiser.
- TikTok's Marketing API advertiser flow returns a long-lived access token with **no refresh token** — when the token is revoked the admin sees a Reconnect prompt in the connector panel. Do not ask users for tokens.
- Most TikTok endpoints take an `advertiser_id` query param. Call `tiktokads_list_advertisers` first to discover authorized IDs.
- The connector's `accessMode` setting (`readonly` or `readwrite`) controls write safety — write methods are blocked when set to `readonly`.

### Empty advertiser list

If `tiktokads_list_advertisers` returns `data.list: []` with `code: 0, message: "OK"`, the OAuth grant succeeded but the user did not authorize any ad accounts on TikTok's consent screen. Almost every Marketing API endpoint needs an `advertiser_id`, so most tools will fail until they reconnect and pick at least one advertiser. Endpoints that don't need an advertiser (e.g. `/user/info/`) will still work. Tell the user to reconnect via the connector settings rather than retrying API calls.

---

## MCP Tools

- `mcp__resources__tiktokads_list_advertisers` - List advertiser/ad accounts authorized for this OAuth token. Args: `resourceId`
- `mcp__resources__tiktokads_list_campaigns` - List campaigns for one advertiser. Args: `resourceId`, `advertiserId`, `campaignIds?`, `campaignName?`, `page?`, `pageSize?`
- `mcp__resources__tiktokads_get_campaign` - Fetch one campaign by ID. Args: `resourceId`, `advertiserId`, `campaignId`
- `mcp__resources__tiktokads_run_report` - Run a Marketing API report. Args: `resourceId`, `advertiserId`, `metrics`, `reportType?`, `dataLevel?`, `dimensions?`, `startDate?`, `endDate?`, `page?`, `pageSize?`
- `mcp__resources__tiktokads_invoke` - Escape hatch for any TikTok Marketing API request under `/open_api/v1.3/`. Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

Prefer the typed tools over `tiktokads_invoke` when they cover the use case. Write methods through `tiktokads_invoke` require the connector's `accessMode` to be `readwrite`.

---

## TypeScript Client

Use the typed helpers for common Marketing API workflows. Use `invoke()` only when a helper does not cover the endpoint.

```typescript
import { tiktokAdsClient } from "./clients";

const advertisers = await tiktokAdsClient.listAdvertisers("fetch-tiktok-advertisers");

if (!advertisers.ok) {
	throw new Error(advertisers.error.message);
}

const campaigns = await tiktokAdsClient.listCampaigns("1234567890", "fetch-tiktok-campaigns", {
	pageSize: 20,
});

if (!campaigns.ok) {
	throw new Error(campaigns.error.message);
}
```

For reports from app code, request only the metrics needed for the UI or report.

```typescript
const report = await tiktokAdsClient.runReport(
	"1234567890",
	["impressions", "clicks", "spend"],
	"fetch-tiktok-report",
	{
		reportType: "BASIC",
		dataLevel: "AUCTION_CAMPAIGN",
		dimensions: ["campaign_id", "stat_time_day"],
		startDate: "2026-04-01",
		endDate: "2026-04-30",
	},
);

if (!report.ok) {
	throw new Error(report.error.message);
}
```

---

## Raw Invoke Rules

`tiktokads_invoke` is a thin API wrapper. Include TikTok-required fields exactly as TikTok expects them. The connector injects the access token automatically; for `/oauth2/advertiser/get/` it also injects `app_id` and `secret`.

For advertiser-scoped endpoints, include `advertiser_id` in `query` or JSON `body` yourself:

```json
{
	"method": "GET",
	"path": "/campaign/get/",
	"query": {
		"advertiser_id": ["1234567890"]
	}
}
```

Direct calls to `/oauth2/access_token` and `/oauth2/refresh_token` are blocked — authentication is handled by the connector.

---

## TikTok API Notes

- Most Marketing API list endpoints accept `filtering` as a JSON-stringified object inside the query (TikTok's convention). The typed `listCampaigns` builder handles this for you; raw `invoke` callers must stringify themselves.
- Reports require both `dimensions` and `metrics` to be JSON-stringified arrays in the query.
- TikTok responses use `{ code, message, data, request_id }` as the envelope. `code: 0` means success; non-zero codes carry a human-readable `message`.
- Access tokens for the advertiser flow are long-lived — there is no refresh. If a call fails with auth errors, ask the user to reconnect the resource.

**Docs**: [TikTok Marketing API](https://business-api.tiktok.com/portal/docs)
