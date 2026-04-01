---
name: using-graphql-connector
description: Executes GraphQL queries and mutations against a configured endpoint using generated clients and MCP tools. Use when doing ANYTHING that touches a GraphQL resource in any way, load this skill.
---

# Major Platform Resource: GraphQL API

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Description field:** Always include a short `description` (~5 words) when calling any resource MCP tool, explaining what the operation does (e.g. "List all user accounts", "Check table schema"). This is displayed to the user in the chat UI.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__graphql_query` — Execute a read-only GraphQL query. Args: `resourceId`, `query`, `variables?`, `operationName?`, `headers?`, `timeoutMs?`
- `mcp__resources__graphql_mutate` — Execute a GraphQL mutation (create, update, delete). Args: `resourceId`, `mutation`, `variables?`, `operationName?`, `headers?`, `timeoutMs?`
- `mcp__resources__graphql_introspect` — Fetch the full GraphQL schema via introspection. Args: `resourceId`

## TypeScript Client

```typescript
import { graphqlClient } from "./clients";

// query(query, invocationKey, options?)
const result = await graphqlClient.query(
	`query GetUsers($limit: Int) {
    users(limit: $limit) {
      id
      name
      email
    }
  }`,
	"list-users",
	{ variables: { limit: 10 }, headers: { "X-Request-ID": "abc-123" } },
);
if (result.ok) {
	const response = result.result;
	// response: { kind: "api", status: number, body: { kind: "json", value: { data: { users: [...] } } } }
}

// mutate(mutation, invocationKey, options?)
const createResult = await graphqlClient.mutate(
	`mutation CreateUser($input: CreateUserInput!) {
    createUser(input: $input) {
      id
      name
    }
  }`,
	"create-user",
	{ variables: { input: { name: "Jane", email: "jane@example.com" } } },
);
if (createResult.ok) {
	console.log(createResult.result.body);
}
```

## Tips

- **Use variables** — always pass dynamic values via the `variables` parameter, never interpolate them into the query string
- **GraphQL returns 200 even on errors** — check `response.body.value.errors` in addition to HTTP status. GraphQL errors are returned in the response body with a 200 status code
- **Use introspection for schema discovery** — run `graphql_introspect` to see available types, queries, and mutations before writing queries
- **Auth headers are automatically injected** — the resource configuration includes the authentication header that you don't need to set manually
- **Per-request headers** — pass `headers` to include additional HTTP headers (e.g. `X-Request-ID`, tenant headers). Per-request headers override the resource's configured auth header if they use the same header name
- Both `query` and `mutate` methods accept the same shape; `mutate` is semantically separated for clarity and MCP read-only enforcement

**Docs**: Refer to the specific GraphQL API's documentation for schema details.
