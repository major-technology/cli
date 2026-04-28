---
name: using-metamarketing-connector
description: Implements Meta (Facebook) Marketing API access for campaigns, ads, insights, and lead forms using generated clients and MCP tools. Use when doing ANYTHING that touches Meta Marketing, Facebook Ads, or Instagram Ads in any way, load this skill.
---

# Major Platform Resource: Meta Marketing

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.json`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-campaign-insights"`, never dynamic values like `` `${date}-insights` ``.

---

## MCP Tools

- `mcp__resources__metamarketing_get` — Make a GET request to any Meta Marketing API endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__metamarketing_get_campaigns` — List campaigns for an ad account. Args: `resourceId`, `adAccountId`, `fields?`, `limit?`, `after?`
- `mcp__resources__metamarketing_get_adsets` — List ad sets for an ad account. Args: `resourceId`, `adAccountId`, `fields?`, `limit?`, `after?`
- `mcp__resources__metamarketing_get_ads` — List ads for an ad account. Args: `resourceId`, `adAccountId`, `fields?`, `limit?`, `after?`
- `mcp__resources__metamarketing_get_insights` — Get performance insights/analytics for an ad account. Args: `resourceId`, `adAccountId`, `fields?`, `datePreset?`, `timeRange?`, `level?`, `limit?`
- `mcp__resources__metamarketing_get_lead_forms` — List lead generation forms for a Facebook page. Args: `resourceId`, `pageId`, `limit?`, `after?`
- `mcp__resources__metamarketing_get_leads` — Get leads submitted to a lead form. Args: `resourceId`, `formId`, `limit?`, `after?`
- `mcp__resources__metamarketing_invoke` — Make any HTTP request (GET/POST/DELETE) for write operations. Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

This client uses the Stripe-style flattened invoke pattern — check `.ok`, then access `.json` directly (no need to dig through `result.body.kind`/`result.body.value`).

```typescript
import { metamarketingClient } from "./clients";

// List campaigns for an ad account
const result = await metamarketingClient.invoke("GET", "/v21.0/act_123456/campaigns", "list-campaigns", {
  query: { fields: "id,name,status,objective,daily_budget", limit: "25" },
});
if (result.ok) {
  const campaigns = result.json.data; // typed as T (default unknown)
}

// Get campaign insights with date filtering
const insights = await metamarketingClient.invoke("GET", "/v21.0/act_123456/insights", "get-insights", {
  query: {
    fields: "campaign_name,impressions,clicks,spend,ctr,cpc",
    date_preset: "last_30d",
    level: "campaign",
  },
});
if (insights.ok) {
  const rows = insights.json.data;
}

// Get leads from a lead form
const leads = await metamarketingClient.invoke("GET", "/v21.0/1234567890/leads", "get-form-leads", {
  query: { fields: "id,created_time,field_data" },
});
if (leads.ok) {
  const leadRecords = leads.json.data;
}

// Create a campaign (requires campaign_management or lead_forms scope preset)
const newCampaign = await metamarketingClient.invoke("POST", "/v21.0/act_123456/campaigns", "create-campaign", {
  body: {
    type: "json",
    value: {
      name: "Summer Sale 2025",
      objective: "OUTCOME_TRAFFIC",
      status: "PAUSED",
      special_ad_categories: [],
    },
  },
});
if (newCampaign.ok) {
  const campaignId = newCampaign.json.id;
}
```

## Tips

- **Ad account IDs use the `act_` prefix** — always include it, e.g. `act_123456789`. The ad account ID is configured per environment as `adAccountId`.
- **Graph API versioning**: All paths include a version prefix like `/v21.0/`. The connector auto-prefixes `/v21.0` if the path starts with `/` but not `/v`.
- **Field selection**: Meta's API returns minimal fields by default. Always pass a `fields` query parameter to request specific fields (e.g., `fields: "id,name,status,insights{impressions,clicks}"`).
- **Pagination**: Meta uses cursor-based pagination. Responses include a `paging` object with `cursors.after` and `cursors.before`. Pass `after` as a query parameter for the next page. Check for `paging.next` to know if more pages exist.
- **Rate limiting**: The Marketing API has rate limits tied to the ad account. Responses include `x-business-use-case-usage` headers. Respect 429 responses and back off.
- **Insights date presets**: Use `date_preset` for common ranges: `today`, `yesterday`, `last_7d`, `last_14d`, `last_30d`, `this_month`, `last_month`, `last_90d`. Or use `time_range` with `{"since":"2025-01-01","until":"2025-01-31"}`.
- **Insights levels**: The `level` parameter controls aggregation: `ad`, `adset`, `campaign`, `account`.
- **Scope tiers build on each other**: `campaign_analytics` (read-only) → `campaign_management` (read/write) → `lead_forms` (everything). Requesting a higher tier implicitly grants all lower-tier permissions.
- **Common endpoints**:
  - `/v21.0/act_{id}/campaigns` — list campaigns
  - `/v21.0/act_{id}/adsets` — list ad sets
  - `/v21.0/act_{id}/ads` — list ads
  - `/v21.0/act_{id}/insights` — account-level insights
  - `/v21.0/{campaign_id}/insights` — campaign-level insights
  - `/v21.0/{page_id}/leadgen_forms` — lead forms for a page
  - `/v21.0/{form_id}/leads` — leads for a form
  - `/v21.0/me/adaccounts` — list ad accounts for the authenticated user

**Docs**: [Meta Marketing API Reference](https://developers.facebook.com/docs/marketing-apis)
