---
name: using-linkedinads-connector
description: Implements LinkedIn Marketing API data access for ad accounts, campaigns, creatives, and analytics using generated clients and MCP tools. Use when doing ANYTHING that touches LinkedIn Ads in any way, load this skill.
---

# Major Platform Resource: LinkedIn Marketing API

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-ad-accounts"`, never dynamic values.

---

## MCP Tools

- `mcp__resources__linkedinads_get` — Make a GET request to any LinkedIn Marketing API endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__linkedinads_list_ad_accounts` — List ad accounts. Args: `resourceId`, `count?`, `start?`
- `mcp__resources__linkedinads_list_campaigns` — List campaigns for an ad account. Args: `resourceId`, `accountId`, `count?`, `start?`
- `mcp__resources__linkedinads_get_campaign_analytics` — Get campaign analytics. Args: `resourceId`, `accountId`, `campaignIds?`, `startDate`, `endDate`, `fields?`
- `mcp__resources__linkedinads_invoke` — Make any HTTP request (POST/PUT/PATCH/DELETE). Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { linkedinAdsClient } from "./clients";

// invoke(method, path, invocationKey, options?)
const result = await linkedinAdsClient.invoke("GET", "/rest/adAccounts", "list-ad-accounts", {
    query: { q: "search", count: "10" },
});
if (result.ok && result.result.status === 200 && result.result.body.kind === "json") {
    const accounts = result.result.body.value.elements;
}
```

---

## Tips

- **LinkedIn Marketing API uses versioned headers** — Include `LinkedIn-Version: YYYYMM` header (the connector handles this automatically). The API version is set in the handler.
- **URN format**: LinkedIn uses URNs like `urn:li:sponsoredAccount:123456` for resource identifiers.
- **Pagination**: Use `start` and `count` query params. Response includes `paging.total` for total count.
- **Rate limits**: Varies by endpoint. Respect `X-RateLimit-Limit` and `X-RateLimit-Remaining` headers.
- **Analytics date format**: Use ISO 8601 dates. The `dateRange` parameter uses `start.day`, `start.month`, `start.year` format.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`

**Docs**: [LinkedIn Marketing API Reference](https://learn.microsoft.com/en-us/linkedin/marketing/)
