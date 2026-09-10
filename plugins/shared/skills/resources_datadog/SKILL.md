---
name: using-datadog-connector
description: Implements Datadog API requests with automatic auth header injection using HTTP proxy and MCP tools. Use when doing ANYTHING that touches a Datadog resource.
---

# Major Platform Resource: Datadog

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use MCP tools or the HTTP proxy.

**Two ways to interact with Datadog:**

1. **MCP tools** (direct, no code needed): Datadog's hosted MCP tools are available through the connector. Use `mcp__resources__list_resources` to discover resources.
2. **HTTP proxy** (Next.js apps): Use `createProxyFetch` from `@major-tech/resource-client/next` to call the Datadog REST API with automatic auth injection.

**Error handling**: Always check response status codes. Datadog returns standard HTTP errors.

## HTTP Proxy Usage

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const ddFetch = createProxyFetch({ resourceId: "your-resource-id" });

// Query metrics
const response = await ddFetch("/api/v1/query?query=avg:system.cpu.user{*}&from=...&to=...");
const data = await response.json();

// List monitors
const monitors = await ddFetch("/api/v1/monitor");

// Search logs
const logs = await ddFetch("/api/v2/logs/events/search", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    filter: { query: "service:my-service status:error", from: "now-1h", to: "now" },
    page: { limit: 25 },
  }),
});
```

## Tips

- **Paths are relative to the region-specific API base URL** (e.g., `api.datadoghq.com` for US1)
- **Auth headers (DD-API-KEY and DD-APPLICATION-KEY) are automatically injected** — never set them manually
- Datadog API is versioned: `/api/v1/...` and `/api/v2/...` — check Datadog docs for the correct version
- **Rate limits**: Datadog enforces rate limits per endpoint; check `X-RateLimit-*` response headers
- **Pagination**: Many list endpoints support `page[limit]` and `page[offset]` or cursor-based pagination
