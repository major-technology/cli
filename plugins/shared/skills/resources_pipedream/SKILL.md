---
name: using-pipedream-connector
description: Implements Pipedream REST API access for managing apps, accounts, workflows, and Connect resources using HTTP proxy MCP tools. Use when doing ANYTHING that touches Pipedream in any way.
---

# Major Platform Resource: Pipedream

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **HTTP Proxy tools**: Use `mcp__resources__http_proxy_get` for read operations and `mcp__resources__http_proxy_invoke` for write operations.

## MCP Tools

The Pipedream connector is proxy-only — use the generic HTTP proxy tools:

- `mcp__resources__http_proxy_get` — Make GET requests to the Pipedream API. Pass the resource ID and either a full URL (`https://api.pipedream.com/v1/...`) or a relative path (`/v1/...`).
- `mcp__resources__http_proxy_invoke` — Make any HTTP method call (POST, PUT, DELETE, etc.) to the Pipedream API.

### Example calls

```
# List apps
http_proxy_get(resourceId: "<id>", url: "/v1/apps")

# Get current user
http_proxy_get(resourceId: "<id>", url: "/v1/users/me")

# List connected accounts for a project
http_proxy_get(resourceId: "<id>", url: "/v1/connect/<project_id>/accounts")

# Create a Connect token
http_proxy_invoke(resourceId: "<id>", method: "POST",
  url: "/v1/connect/<project_id>/tokens",
  body: { type: "json", value: { external_user_id: "user-123" } })
```

## Tips

- The Pipedream API uses OAuth2 Client Credentials authentication — the connector handles token exchange automatically.
- All API paths start with `/v1/`. You can use relative paths (e.g. `/v1/users/me`) — the proxy prepends the base URL.
- For Connect API endpoints, include the project ID in the path: `/v1/connect/{project_id}/...`
- Pass `X-PD-Environment: development` or `X-PD-Environment: production` as a header when calling Connect API endpoints that are environment-scoped.
- **Docs**: [Pipedream REST API](https://pipedream.com/docs/rest-api/) and [Connect API](https://pipedream.com/docs/connect/api-ref)
