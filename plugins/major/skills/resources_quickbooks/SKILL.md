---
name: using-quickbooks-connector
description: Implements QuickBooks Online accounting data access for customers, invoices, items, accounts, vendors, bills, and payments using generated clients and MCP tools. Use when doing ANYTHING that touches QuickBooks in any way, load this skill.
---

# Major Platform Resource: QuickBooks Online

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

- `mcp__resources__quickbooks_query` — Execute a QuickBooks query using SQL-like syntax. Args: `resourceId`, `query`, `timeoutMs?`
- `mcp__resources__quickbooks_get` — Get a specific entity by type and ID. Args: `resourceId`, `entityType`, `entityId`
- `mcp__resources__quickbooks_invoke` — Make any HTTP request to the QuickBooks API, including write operations. Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

```typescript
import { qbClient } from "./clients";

// Query entities using SQL-like syntax
const result = await qbClient.query("SELECT * FROM Customer WHERE DisplayName LIKE 'A%'", "search-customers");
if (result.ok && result.result.body.kind === "json") {
	const data = result.result.body.value;
}

// Generic invoke for any operation
const invoice = await qbClient.invoke("GET", "/invoice/123", "get-invoice");

// Create an invoice
const newInvoice = await qbClient.invoke("POST", "/invoice", "create-invoice", {
	body: {
		type: "json",
		value: {
			Line: [{ Amount: 100.0, DetailType: "SalesItemLineDetail" }],
			CustomerRef: { value: "1" },
		},
	},
});
```

## Tips

- **QuickBooks Query Language** is SQL-like but has limitations:
  - No JOINs, no OR in WHERE clauses, no GROUP BY
  - Use `LIKE` with `%` for wildcard matching
  - Only filterable properties can be used in WHERE clauses
  - Example: `SELECT * FROM Invoice WHERE TotalAmt > '100.00'`
- **Common entity types**: Customer, Invoice, Item, Account, Vendor, Bill, Payment, Estimate, PurchaseOrder, SalesReceipt, CreditMemo, Employee
- **All API paths are relative** to `/v3/company/{realmId}` — the realmId is handled automatically
- **Rate limit**: 500 requests per minute per realm. Respect throttling headers.
- Response structure: `{ kind: "api", status: number, body: { kind: "json", value: {...} } }`
- **Updates require SyncToken**: When updating entities, include the current `SyncToken` from the entity to prevent conflicts

**Docs**: [QuickBooks Online API Reference](https://developer.intuit.com/app/developer/qbo/docs/api/accounting/all-entities/account)
