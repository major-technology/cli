---
name: using-googlesearchconsole-connector
description: Implements Google Search Console data access for search analytics, sitemaps, sites, and URL inspection using generated clients and MCP tools. Use when doing ANYTHING that touches Google Search Console or SEO search data, load this skill.
---

# Major Platform Resource: Google Search Console

Google Search Console is an OAuth-only connector. It accesses verified Search Console properties for the signed-in Google user. Do not describe or use it as an API-key/public URL testing connector.

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-top-queries"`, never dynamic values like `` `${date}-report` ``.

---

## MCP Tools

- `mcp__resources__googlesearchconsole_query_analytics` — Query search analytics data (clicks, impressions, CTR, position). Args: `resourceId`, `startDate`, `endDate`, `dimensions?`, `searchType?`, `dimensionFilterGroups?`, `rowLimit?`, `startRow?`
- `mcp__resources__googlesearchconsole_list_sites` — List all verified sites in Search Console. Args: `resourceId`
- `mcp__resources__googlesearchconsole_get_site` — Get info about a specific site. Args: `resourceId`, `siteUrl`
- `mcp__resources__googlesearchconsole_list_sitemaps` — List sitemaps for a site. Args: `resourceId`, `siteUrl`
- `mcp__resources__googlesearchconsole_invoke` with `operation: "getSitemap"` — Get one sitemap. Args: `resourceId`, `operation`, `siteUrl`, `feedpath`
- `mcp__resources__googlesearchconsole_inspect_url` — Inspect a URL's index status. Args: `resourceId`, `siteUrl`, `inspectionUrl`
- `mcp__resources__googlesearchconsole_invoke` — Execute any Search Console operation. Args: `resourceId`, `operation`, + operation-specific args

## TypeScript Client

```typescript
import { gscClient } from "./clients";

// queryAnalytics(startDate, endDate, invocationKey, options?)
const result = await gscClient.queryAnalytics("2024-01-01", "2024-01-31", "top-queries", {
    dimensions: ["query", "page"],
    rowLimit: 100,
});

// listSites(invocationKey)
const sites = await gscClient.listSites("list-gsc-sites");

// listSitemaps(siteUrl, invocationKey)
const sitemaps = await gscClient.listSitemaps("https://example.com/", "list-sitemaps");

// getSitemap(siteUrl, feedpath, invocationKey)
const sitemap = await gscClient.getSitemap(
    "https://example.com/",
    "https://example.com/sitemap.xml",
    "get-sitemap",
);

// inspectUrl(siteUrl, inspectionUrl, invocationKey)
const inspection = await gscClient.inspectUrl("https://example.com/", "https://example.com/page", "inspect-page");
```

## Tips

- **OAuth only**: This connector works with properties the authenticated Google user can access. Use `list_sites` first to discover verified properties.
- **Site URL formats**: Use either `https://example.com/` (URL prefix) or `sc-domain:example.com` (domain property). Pass the exact value Search Console returns.
- **Data delay**: GSC data is typically 2-3 days behind real-time. Don't expect yesterday's data to be complete.
- **Dimensions**: Available: `query`, `page`, `country`, `device`, `date`, `searchAppearance`. Include `date` to get daily breakdowns.
- **Search types**: `web` (default), `image`, `video`, `news`, `discover`, `googleNews`.
- **Row limit**: Max 25,000 rows per request. Use `startRow` for pagination.
- **Date format**: Use `YYYY-MM-DD` format for start/end dates.
- **Filter operators**: `contains`, `equals`, `notContains`, `notEquals`, `includingRegex`, `excludingRegex`.
- **URL Inspection**: Inspects one URL at a time. Rate-limited to 600 QPM/site.

**Docs**: [Search Console API](https://developers.google.com/webmaster-tools/v1/api_reference_index) | [Search Analytics](https://developers.google.com/webmaster-tools/v1/searchanalytics)
