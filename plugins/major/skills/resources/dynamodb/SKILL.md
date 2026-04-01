---
name: using-dynamodb-connector
description: Implements DynamoDB queries, scans, and CRUD operations using generated clients and MCP tools. Use when doing ANYTHING that touches DynamoDB, DDB, or Dynamo in any way, load this skill.
---

# Major Platform Resource: DynamoDB

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__dynamodb_list_tables` — List all accessible tables. Args: `resourceId`, `limit?`
- `mcp__resources__dynamodb_describe_table` — Get table schema, keys, indexes, throughput. Args: `resourceId`, `tableName`
- `mcp__resources__dynamodb_scan` — Read-only scan with optional filter. Args: `resourceId`, `tableName`, `limit?`, `filterExpression?`, `projectionExpression?`

## TypeScript Client

```typescript
import { myDynamoClient } from "./clients";

// invoke<C>(command, params, invocationKey)
const result = await myDynamoClient.invoke(
	"Query",
	{
		TableName: "orders",
		KeyConditionExpression: "user_id = :uid",
		ExpressionAttributeValues: { ":uid": { S: userId } },
		Limit: 20,
		ScanIndexForward: false,
	},
	"fetch-user-orders",
);

if (result.ok) {
	console.log(result.result.data.Items);
}
```

## Tips

- **Prefer Query over Scan** — Scan reads every item in the table and is expensive at scale
- Use `describe_table` to understand key schema and indexes before writing queries
- DynamoDB attribute values use marshall format (`{ S: "string" }`, `{ N: "123" }`, `{ BOOL: true }`)
- TypeScript client supports all DynamoDB commands: `GetItem`, `PutItem`, `Query`, `Scan`, `UpdateItem`, `DeleteItem`, `BatchGetItem`, `BatchWriteItem`

**Docs**: [DynamoDB Developer Guide](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/)
