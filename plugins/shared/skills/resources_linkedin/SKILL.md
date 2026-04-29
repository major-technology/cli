---
name: using-linkedin-connector
description: Implements LinkedIn Marketing API access for ad accounts, campaigns, creatives, and ad analytics using generated clients and MCP tools. Use when doing ANYTHING that touches the LinkedIn Marketing API — ad accounts, campaigns, creatives, or ad analytics — in any way, load this skill.
---

# Major Platform Resource: LinkedIn Marketing API

Reference: https://learn.microsoft.com/en-us/linkedin/marketing/

## Common: Interacting with Resources

**Security**: Never connect directly to LinkedIn APIs with raw credentials. Never put OAuth tokens in code, logs, env vars, prompts, or user-visible output. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__linkedin_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and exact signatures before writing app code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** - use descriptive literals like `"fetch-linkedin-campaigns"`, never dynamic values like `` `${accountId}-campaigns` ``.

---

## Scope And Permissions

The v1 LinkedIn Marketing API connector exposes the **Advertising API** surface. Lead Sync and Conversions API will ship as separate Major connectors when LinkedIn approval lands; do not assume they are reachable through this resource.

Available scope sets:

- Read only: `r_ads`, `r_ads_reporting`
- Read + write: `r_ads`, `r_ads_reporting`, `rw_ads`
- Custom: any subset of the above (admins pick at connector creation time)

If LinkedIn refresh fails because the refresh token is revoked or expired, the connector call will fail and the admin will see a Reconnect prompt in the connector panel. Do not ask users for tokens — every LinkedIn call goes through the connector.

---

## MCP Tools

- `mcp__resources__linkedin_list_ad_accounts` - List accessible LinkedIn ad accounts. Args: `resourceId`, `status?`, `pageSize?`, `pageToken?`
- `mcp__resources__linkedin_list_campaigns` - List campaigns for an ad account. Args: `resourceId`, `adAccountId`, `status?`, `pageSize?`, `pageToken?`
- `mcp__resources__linkedin_list_creatives` - List creatives for an ad account or campaigns. Args: `resourceId`, `adAccountId`, `campaignIds?`, `pageSize?`, `pageToken?`
- `mcp__resources__linkedin_get_ad_analytics` - Fetch ad analytics. Args: `resourceId`, `adAccountId`, `dateRange`, `pivot`, `metrics?`
- `mcp__resources__linkedin_invoke` - Escape hatch for LinkedIn Marketing API requests. Args: `resourceId`, `method`, `path`, `query?`, `body?`

Prefer typed list and analytics tools over `linkedin_invoke` when they cover the use case.

---

## TypeScript Client

Use the typed helpers for common Advertising API workflows. Use `invoke()` only when a helper does not cover the endpoint.

```typescript
import { linkedinClient } from "./clients";

const accounts = await linkedinClient.listAdAccounts("fetch-linkedin-ad-accounts", {
	pageSize: 25,
});

if (!accounts.ok) {
	throw new Error(accounts.error.message);
}

const campaigns = await linkedinClient.listCampaigns("123456", "fetch-linkedin-campaigns", {
	status: ["ACTIVE"],
	pageSize: 25,
});

if (!campaigns.ok) {
	throw new Error(campaigns.error.message);
}
```

For analytics from app code, request only the fields needed for the UI or report.

```typescript
const analytics = await linkedinClient.getAdAnalytics(
	"123456",
	{ start: "2026-04-01", end: "2026-04-30" },
	"fetch-linkedin-analytics",
	{
		pivot: "CAMPAIGN",
		metrics: ["impressions", "clicks", "costInLocalCurrency"],
	},
);

if (!analytics.ok) {
	throw new Error(analytics.error.message);
}
```

---

## LinkedIn API Notes

- Use numeric IDs returned by LinkedIn tools when available. The connector handles LinkedIn REST headers and URN formatting for typed tools.
- Ad account and campaign IDs may appear either as plain IDs or URNs in LinkedIn responses. Preserve IDs exactly unless the generated client documents a normalized field.
- Analytics date ranges: pass `dateRange: { start: "YYYY-MM-DD", end: "YYYY-MM-DD" }`. The connector converts to LinkedIn's internal date-object format internally.
- Analytics metrics can be expensive. Keep date ranges and fields narrow.
- LinkedIn Marketing API responses commonly return `{ elements, paging }`.

**Docs**: [LinkedIn Marketing API](https://learn.microsoft.com/en-us/linkedin/marketing/)
