---
name: using-bigquery-connector
description: Implements BigQuery dataset exploration, SQL queries, and table operations using generated clients and MCP tools. Use when doing ANYTHING that touches BigQuery or BQ in any way, load this skill.
---

# Major Platform Resource: BigQuery

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

- `mcp__resources__bigquery_list_datasets` — List all datasets in the project. Args: `resourceId`
- `mcp__resources__bigquery_list_tables` — List tables in a dataset. Args: `resourceId`, `datasetId`
- `mcp__resources__bigquery_describe_table` — Get schema and metadata for a table. Args: `resourceId`, `datasetId`, `tableId`
- `mcp__resources__bigquery_query` — Execute read-only SQL (SELECT only). Args: `resourceId`, `statement`

## TypeScript Client

```typescript
import { bqClient } from "./clients";

// query(sql, params?, invocationKey, options?)
const result = await bqClient.query(
	"SELECT * FROM `project.dataset.table` WHERE created > @cutoff",
	{ cutoff: "2024-01-01" },
	"recent-records",
	{ maxResults: 1000 },
);

// Other methods: listDatasets, listTables, getTable, insertRows, createTable
```

## Tips

- **Be cost-aware** — BigQuery charges per bytes scanned. Use `SELECT specific_columns` instead of `SELECT *`. Use `LIMIT` during exploration.
- Use `list_datasets` → `list_tables` → `describe_table` to understand data structure before querying
- Use `maxResults` option for pagination of large result sets
- Named parameters use `@param` syntax in queries

**Docs**: [BigQuery Documentation](https://cloud.google.com/bigquery/docs)
