---
name: using-googlesheets-connector
description: Implements Google Sheets reading, writing, formatting, and batch operations using generated clients and MCP tools. Use when doing ANYTHING that touches Google Sheets or gsheets in any way, load this skill.
---

# Major Platform Resource: Google Sheets

## Setting Up a Google Sheets Connector

Google Sheets requires a two-step setup: (1) OAuth authentication, (2) spreadsheet selection.

### When the user asks you to set up Google Sheets or connect a spreadsheet:

1. Call `mcp__resource-setup__request-resource-setup` with `subtype: "googlesheets"` — this prompts the user to authenticate with Google
2. After setup completes, call `mcp__resource-setup__request-resource-update` with the returned `resourceId` and `message: "Please select your spreadsheet. Click 'Configure Resource' below, then use the spreadsheet picker to choose your sheet."` — this prompts them to select their spreadsheet
3. Once both steps complete, the resource is ready to use

### When the user sends a Google Sheets link:

If the user shares a Google Sheets URL (e.g., `https://docs.google.com/spreadsheets/d/...`), you cannot connect to it directly via the URL. Explain that they need to set up a Google Sheets connector:

1. Tell them: "To connect to this spreadsheet, we need to set up a Google Sheets connector. This involves authenticating with Google and then selecting your spreadsheet."
2. Follow the setup flow above (steps 1-3)
3. After setup, the resource will be bound to the spreadsheet they select — remind them to pick the correct one

### When a Google Sheets resource exists but has no spreadsheet selected:

If you call a Google Sheets MCP tool and get an error indicating no spreadsheet is configured, use `mcp__resource-setup__request-resource-update` to prompt the user to select one.

---

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

- `mcp__resources__googlesheets_get_metadata` — Get spreadsheet metadata (title, locale, sheet info). Args: `resourceId`
- `mcp__resources__googlesheets_get_values` — Read cell values from a range. Args: `resourceId`, `range`
- `mcp__resources__googlesheets_list_sheets` — List all sheets (tabs) with properties. Args: `resourceId`

## TypeScript Client

**Prefer helper methods over raw `invoke()`:**

```typescript
import { sheetsClient } from "./clients";

// Read values
const values = await sheetsClient.getValues("Sheet1!A1:D10", "fetch-data");

// Append rows
await sheetsClient.appendValues("Sheet1!A:D", [["John", "Doe", "john@example.com", "2024-01-15"]], "append-row", {
	valueInputOption: "USER_ENTERED",
});

// Update values
await sheetsClient.updateValues(
	"Sheet1!A1:B2",
	[
		["Name", "Email"],
		["Jane", "jane@ex.com"],
	],
	"update-cells",
);

// Batch operations
await sheetsClient.batchGetValues(["Sheet1!A1:B5", "Sheet2!A1:C3"], "batch-read");

// Formatting via batchUpdate
await sheetsClient.batchUpdate(
	[
		{
			repeatCell: {
				range: { sheetId: 0, startRowIndex: 0, endRowIndex: 1 },
				cell: { userEnteredFormat: { textFormat: { bold: true } } },
				fields: "userEnteredFormat.textFormat.bold",
			},
		},
	],
	"bold-header",
);
```

## Tips

- **Use helper methods** (`getValues`, `appendValues`, `updateValues`, `batchGetValues`, `batchUpdateValues`, `batchUpdate`) over raw `invoke()` when possible
- Each resource is bound to a single spreadsheet — the spreadsheet ID is automatically included
- For raw `invoke()`, paths are relative to `/v4/spreadsheets/{spreadsheetId}` (e.g., `/values/Sheet1!A1:D10`)
- Use `valueInputOption: "USER_ENTERED"` to let Sheets parse formulas and dates

**Docs**: [Google Sheets API Reference](https://developers.google.com/workspace/sheets/api/reference/rest)
