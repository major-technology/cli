---
name: using-neo4j-connector
description: Implements Neo4j Cypher queries, graph traversal, and node/relationship operations using generated clients and MCP tools. Use when doing ANYTHING that touches Neo4j or Cypher in any way, load this skill.
---

# Major Platform Resource: Neo4j

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

- `mcp__resources__neo4j_query` — Execute read-only Cypher (MATCH/CALL/SHOW/WITH/RETURN only). Args: `resourceId`, `cypher`, `params?`, `maxResults?`
- `mcp__resources__neo4j_list_node_labels` — List all node labels. Args: `resourceId`
- `mcp__resources__neo4j_list_relationship_types` — List all relationship types. Args: `resourceId`
- `mcp__resources__neo4j_describe_schema` — Get constraints and indexes. Args: `resourceId`

## TypeScript Client

```typescript
import { graphDbClient } from "./clients";

// invoke(cypher, params?, invocationKey, timeoutMs?)
// Uses named parameters: $paramName
const result = await graphDbClient.invoke(
	"MATCH (u:User {email: $email})-[:PURCHASED]->(p:Product) RETURN u, p LIMIT 10",
	{ email: "jane@example.com" },
	"fetch-user-purchases",
);
if (result.ok) {
	const { records, keys } = result.result;
	for (const record of records) {
		console.log(record["u"].properties.name);
	}
}
```

## Response Shape

Graph types are converted to plain objects with a `_type` discriminator:

| Neo4j Type   | Shape                                                                                    |
| ------------ | ---------------------------------------------------------------------------------------- |
| Node         | `{ _type: "node", _id, labels, properties }`                                             |
| Relationship | `{ _type: "relationship", _id, _startNodeId, _endNodeId, relationshipType, properties }` |
| Path         | `{ _type: "path", nodes[], relationships[] }`                                            |

## Tips

- Uses **named parameters** (`$email`) not positional — pass `undefined` when no params needed
- MCP query tool is read-only; use the TypeScript client for CREATE/MERGE/DELETE operations
- Use `list_node_labels` and `list_relationship_types` to explore the graph structure before querying

**Docs**: [Neo4j Cypher Manual](https://neo4j.com/docs/cypher-manual/current/)
