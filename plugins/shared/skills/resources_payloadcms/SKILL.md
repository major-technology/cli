---
name: using-payloadcms-connector
description: Executes GraphQL queries and mutations against a Payload CMS instance using generated clients and MCP tools. Use when doing ANYTHING that touches Payload CMS in any way, load this skill.
---

# Major Platform Resource: Payload CMS

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `response.ok` before accessing `response.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-posts"`, never dynamic values like `` `${date}-posts` ``.

---

## MCP Tools

- `mcp__resources__payloadcms_query` — Execute a read-only GraphQL query against the Payload CMS endpoint. Args: `resourceId`, `query`, `variables?`, `operationName?`
- `mcp__resources__payloadcms_mutate` — Execute a GraphQL mutation against the Payload CMS endpoint. Args: `resourceId`, `mutation`, `variables?`, `operationName?`
- `mcp__resources__payloadcms_introspect` — Fetch the full GraphQL schema via introspection. Args: `resourceId`

## TypeScript Client

The client exposes `query()` and `mutate()` methods that return the raw API invoke response.

### Reading data

```typescript
import { payloadcmsClient } from "./clients";

// List all posts
const response = await payloadcmsClient.query(
  `{ Posts { docs { id title status } } }`,
  "list-posts"
);

if (response.ok) {
  const data = response.result.body.value as { data: { Posts: { docs: Array<{ id: string; title: string }> } } };
  console.log(data.data.Posts.docs);
}

// Get a single post by ID
const response = await payloadcmsClient.query(
  `query GetPost($id: String!) { Post(id: $id) { id title content } }`,
  "get-post",
  { variables: { id: "abc123" } }
);
```

### Writing data

```typescript
// Create a new post
const response = await payloadcmsClient.mutate(
  `mutation CreatePost($data: mutationPostInput!) {
    createPost(data: $data) { id title }
  }`,
  "create-post",
  { variables: { data: { title: "My Post", content: "Hello world" } } }
);

// Update an existing post
const response = await payloadcmsClient.mutate(
  `mutation UpdatePost($id: String!, $data: mutationPostUpdateInput!) {
    updatePost(id: $id, data: $data) { id title }
  }`,
  "update-post",
  { variables: { id: "abc123", data: { title: "Updated Title" } } }
);
```

## Tips

- **GraphQL returns 200 even on errors** — always check `response.result.body.value.errors` for GraphQL-level errors after confirming `response.ok`.
- **Use introspection** (`payloadcms_introspect`) to discover available collections, types, and fields before writing queries.
- **Auth header is auto-constructed** from the collection slug + API key. You never need to set auth headers manually.
- **Collection slugs become GraphQL type names** — spaces and special characters are removed. A collection with slug `blog-posts` becomes `BlogPosts` (singular `BlogPost` for single-doc queries).
- **Pagination pattern**: Payload CMS returns `{ docs: [...], totalDocs, limit, page, totalPages, hasNextPage, hasPrevPage }`.
- **Common query patterns**:
  - List: `{ Posts { docs { id title } } }`
  - Single: `{ Post(id: "xxx") { id title } }`
  - With pagination: `{ Posts(limit: 10, page: 2) { docs { id } totalDocs hasNextPage } }`
- **Docs**: https://payloadcms.com/docs/graphql/overview and https://payloadcms.com/docs/authentication/api-keys
