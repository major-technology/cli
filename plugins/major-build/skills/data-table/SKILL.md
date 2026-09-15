---
name: using-frontend-data-table
description: Creates frontend table views. Use whenever creating any type of table UI on the frontend. Use when doing ANYHTHING that involves tables (using tables to display data).
---

# DataTable

A composable, full-featured DataTable built on TanStack Table v8. Installed via the shadcn registry — code lives locally in the project.

## Installation

Install via the shadcn registry:

```bash
npx shadcn@latest add https://cdn.major.build/shadcn/data-table.json
```

This copies the data table source code into `components/data-table/` in the project. All imports use the project's path alias (typically `@/components/data-table`).

## Is a Table the Right UI?

Before building a table, evaluate whether the data actually fits a tabular layout. Check these signals:

**Use a table when:**

- Each record has 3+ distinct fields (e.g. name, email, role, status, created date)
- Users need to compare values across rows (scanning columns)
- The data benefits from sorting, filtering, or pagination
- The page is operational — users manage, edit, or act on records

**Don't use a table when:**

- Items are media-heavy (images, thumbnails, previews are the primary content)
- Each item has only 1–2 fields — a simple list or card grid is better
- The layout is a feed, timeline, or activity log with variable-length content
- Visual presentation matters more than data density (e.g. product showcase, profile cards)

**When ambiguous** (e.g. "show me a list of projects" — could be cards or a table), use `AskUserQuestion` to ask the user whether they want a data table or a card/list layout. Briefly explain the trade-off: tables are better for dense, scannable data with actions; cards are better for visual, media-rich items.

If a table is not the right fit, do not use this skill — build the appropriate UI directly instead.

## Feature Selection

Every table includes these features by default — do not ask the user about them:

- **Global search** — Search across all columns with debounced input
- **Column filters** — Dynamic per-column filters (select, text, number, date, boolean)
- **Column sorting** — Click column headers to sort

Before implementing a table, use the `AskUserQuestion` tool to ask the user which additional features they need.

First, ask a single-select question for the pagination strategy:

- **Offset pagination** — Page navigation with page size selector
- **Infinite scroll** — Load more rows automatically as the user scrolls
- **None** — No pagination (all data rendered at once)

Then, ask a multi-select question for optional features:

- **Column visibility** — Toggle which columns are visible
- **Column reordering** — Drag-and-drop to rearrange columns
- **Column resizing** — Drag column borders to resize
- **Row actions** — Three-dot dropdown menu per row (edit, delete, etc.)
- **Row selection** — Checkbox selection with bulk action bar
- **Expandable rows** — Click to expand rows with detail content
- **Data export** — Export current page or all data as CSV
- **Sticky header** — Pin header while scrolling the table body

Only compose the features the user selects. Do not add features they did not ask for. If the user rejects or dismisses the question, infer the most reasonable set of features from context (the data source, dataset size, and what the user described) and proceed without asking again.

**Virtualization:** Do not ask the user about virtualization. Enable it automatically when the page size exceeds 200 rows or when infinite scroll is selected, since those scenarios render enough rows to benefit from it.

## Version Check

When the user asks to modify, update, or add features to an existing table, check whether the installed version is current before making changes:

1. Read the local `DATA_TABLE_VERSION` constant from the project's `data-table/constants.ts`.
2. Fetch `https://cdn.major.build/shadcn/data-table.json` and read the `version` field from the JSON.
3. If the remote version is newer than the local version, inform the user and suggest updating:
   - Run `npx shadcn@latest add https://cdn.major.build/shadcn/data-table.json` to pull the latest code.
   - If the user confirms the update, verify that any existing backend endpoints (API routes, resource queries) still work correctly with the updated table code. Fix any breaking changes.
   - Review the changelog between versions. If the new version introduced features that are relevant to the user's table, suggest incorporating them using `AskUserQuestion`. Only suggest features that make sense for their use case — do not push everything.
4. If versions match, proceed directly with the requested changes.

## Quick Start — Auto Mode

Pass `onLoadRows` and the table manages ALL internal state — data, loading, sorting, filtering, pagination.

```tsx
import { DataTable, DataTableContent, DataTablePagination, buildRequestSearchParams } from "@/components/data-table";
import type { ColumnDef, DataTableResponse } from "@/components/data-table";

interface User {
	id: string;
	name: string;
	email: string;
}

const columns: ColumnDef<User, unknown>[] = [
	{ accessorKey: "name", header: "Name" },
	{ accessorKey: "email", header: "Email" },
];

async function loadUsers(params) {
	const res = await fetch(`/api/users?${buildRequestSearchParams(params)}`);
	if (!res.ok) {
		return { success: false, error: { code: String(res.status), message: "Failed to fetch" } };
	}
	return res.json(); // Must return DataTableResponse<User>
}

export default function UsersPage() {
	return (
		<DataTable onLoadRows={loadUsers} columns={columns}>
			<DataTableContent />
			<DataTablePagination />
		</DataTable>
	);
}
```

## Architecture

**Compound components via context.** The root `<DataTable>` creates a TanStack Table instance and provides it through React Context. Sub-components consume the context — compose only the pieces you need.

**Two modes:**

1. **Auto mode** — Pass `onLoadRows`. The table calls it with `DataTableRequestParams` whenever sorting, filtering, or pagination changes. All data fetching and state management is handled internally.
2. **Static mode** — Pass `data` directly. Sorting, filtering, and pagination happen client-side by default. Optionally pass controlled state props (`sorting` + `onSortingChange`, etc.) for server-controlled static mode.

**Composable.** Every sub-component is optional. Mix and match `<DataTablePagination>`, `<DataTableInfiniteScroll>`, `<DataTableToolbar>`, `<DataTableExport>`, etc.

---

## Composition Guide

### Minimal table — just content

```tsx
<DataTable columns={columns} data={rows}>
	<DataTableContent />
</DataTable>
```

### With pagination

```tsx
<DataTable onLoadRows={loadRows} columns={columns}>
	<DataTableContent />
	<DataTablePagination />
</DataTable>
```

### With infinite scroll

```tsx
<DataTable onLoadRows={loadRows} columns={columns}>
	<DataTableContent />
	<DataTableInfiniteScroll />
</DataTable>
```

### With toolbar (search + filters + column toggle + export)

```tsx
<DataTable onLoadRows={loadRows} columns={columns}>
	<DataTableToolbar>
		<DataTableSearch placeholder="Search users..." />
		<DataTableFilters filters={filterDefinitions} />
		<div className="flex-1" />
		<DataTableExport
			formatRows={(rows) => rowsToCsv(rows)}
			onExportAll={async () => downloadFile("/api/users/export")}
		/>
		<DataTableColumnToggle />
	</DataTableToolbar>
	<DataTableContent />
	<DataTablePagination />
</DataTable>
```

### With row selection + bulk actions

```tsx
<DataTable onLoadRows={loadRows} columns={[createSelectColumn<User>(), ...baseColumns]} getRowId={(u) => u.id}>
	<DataTableRowSelection<User>>
		{(rows) => <button onClick={() => handleBulkDelete(getSelectedRows(rows))}>Delete ({rows.length})</button>}
	</DataTableRowSelection>
	<DataTableContent />
	<DataTablePagination />
</DataTable>
```

### With row actions

```tsx
const columns = [
	...baseColumns,
	createActionsColumn<User>((row) => [
		{ label: "Edit", onSelect: (r) => router.push(`/users/${r.id}/edit`) },
		{ label: "View details", onSelect: (r) => router.push(`/users/${r.id}`) },
		{ type: "separator" },
		{
			label: "Delete",
			variant: "destructive",
			onSelect: async (r, { removeRow }) => {
				await handleDelete(r.id);
				removeRow(r.id);
			},
		},
	]),
];

<DataTable onLoadRows={loadRows} columns={columns}>
	<DataTableContent />
	<DataTablePagination />
</DataTable>;
```

### With expandable rows

```tsx
<DataTable onLoadRows={loadRows} columns={[createExpandColumn<User>(), ...baseColumns]}>
	<DataTableContent renderExpandedRow={(row) => <UserDetail user={row.original} />} />
	<DataTablePagination />
</DataTable>
```

### With virtualization (large datasets)

Virtualization renders only visible rows. Requires `maxHeight` to define the scroll viewport. Built-in infinite scroll triggers `loadNextPage` automatically when nearing the bottom.

```tsx
<DataTable onLoadRows={loadRows} columns={columns}>
	<DataTableContent virtualized maxHeight={600} estimateRowHeight={48} overscan={10} />
</DataTable>
```

### With sticky header

```tsx
<DataTable onLoadRows={loadRows} columns={columns}>
	<DataTableContent stickyHeader maxHeight={500} />
	<DataTablePagination />
</DataTable>
```

### Kitchen sink

```tsx
const columns = [
	createSelectColumn<User>(),
	createExpandColumn<User>(),
	...baseColumns,
	createActionsColumn<User>((row) => [
		{ label: "Edit", onSelect: (r) => router.push(`/users/${r.id}/edit`) },
		{ type: "separator" },
		{
			label: "Delete",
			variant: "destructive",
			onSelect: async (r, { removeRow }) => {
				await api.deleteUser(r.id);
				removeRow(r.id);
			},
		},
	]),
];

<DataTable
	onLoadRows={loadRows}
	columns={columns}
	getRowId={(u) => u.id}
	pageSize={50}
	enableColumnReordering
	enableColumnResizing
	onError={(err) => toast.error(err.message)}
>
	<DataTableToolbar>
		<DataTableSearch placeholder="Search..." />
		<DataTableFilters filters={filterDefinitions} />
		<div className="flex-1" />
		<DataTableExport
			formatRows={(rows) => rowsToCsv(rows)}
			onExportAll={async () => downloadFile("/api/users/export")}
		/>
		<DataTableColumnToggle />
	</DataTableToolbar>
	<DataTableRowSelection<User>>
		{(rows) => <button onClick={() => handleDelete(rows)}>Delete ({rows.length})</button>}
	</DataTableRowSelection>
	<DataTableContent renderExpandedRow={(row) => <UserDetail user={row.original} />} stickyHeader maxHeight={600} />
	<DataTablePagination />
</DataTable>;
```

**Note:** The actions column is automatically pinned to the right edge with a sticky position. When the table overflows horizontally, the actions column stays visible with a subtle left shadow. It is excluded from column drag-and-drop reordering.

### Static mode — client-side data

All sorting, filtering, and pagination happens in the browser. No server calls.

```tsx
<DataTable columns={columns} data={allRows}>
	<DataTableToolbar>
		<DataTableSearch />
		<DataTableFilters filters={filterDefinitions} />
	</DataTableToolbar>
	<DataTableContent />
	<DataTablePagination />
</DataTable>
```

### Static mode with server-controlled state

Pass `data` but also provide controlled state props. The table switches to manual mode when it sees `sorting + onSortingChange`, `pagination + onPaginationChange`, etc.

```tsx
const [sorting, setSorting] = useState<SortingState>([]);
const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 50 });

// Fetch data whenever sorting/pagination changes
const { data, totalCount, isLoading } = useFetchUsers({ sorting, pagination });

<DataTable
	columns={columns}
	data={data}
	isLoading={isLoading}
	sorting={sorting}
	onSortingChange={setSorting}
	pagination={pagination}
	onPaginationChange={setPagination}
	rowCount={totalCount}
>
	<DataTableContent />
	<DataTablePagination />
</DataTable>;
```

---

## Component Reference

### `<DataTable>` — Root

Creates the TanStack Table instance and provides context. All sub-components must be children.

**Auto mode props** (pass `onLoadRows`):

| Prop          | Type                                                 | Default | Description                                                              |
| ------------- | ---------------------------------------------------- | ------- | ------------------------------------------------------------------------ |
| `onLoadRows`  | `DataTableLoadRowsFn<TData>`                         | —       | Promise-returning load function — enables auto mode                      |
| `initialData` | `PaginatedResponse<TData>`                           | —       | Pre-loaded first page for SSR — renders immediately, skips initial fetch |
| `onError`     | `(error: { code: string; message: string }) => void` | —       | Called when `onLoadRows` returns an error response                       |

**Static mode props** (pass `data`):

| Prop                                      | Type                                | Default | Description                                 |
| ----------------------------------------- | ----------------------------------- | ------- | ------------------------------------------- |
| `data`                                    | `TData[]`                           | —       | Row data to display                         |
| `isLoading`                               | `boolean`                           | `false` | Show loading state                          |
| `sorting` / `onSortingChange`             | `SortingState` / `OnChangeFn`       | —       | Controlled sorting (implies manual mode)    |
| `columnFilters` / `onColumnFiltersChange` | `ColumnFiltersState` / `OnChangeFn` | —       | Controlled filtering (implies manual mode)  |
| `globalFilter` / `onGlobalFilterChange`   | `string` / `OnChangeFn`             | —       | Controlled search                           |
| `pagination` / `onPaginationChange`       | `PaginationState` / `OnChangeFn`    | —       | Controlled pagination (implies manual mode) |
| `rowCount`                                | `number`                            | —       | Total row count for server-side pagination  |

**Shared props** (both modes):

| Prop                                            | Type                               | Default | Description                       |
| ----------------------------------------------- | ---------------------------------- | ------- | --------------------------------- |
| `columns`                                       | `ColumnDef<TData>[]`               | —       | Column definitions (required)     |
| `getRowId`                                      | `(row: TData) => string`           | —       | Custom row ID extractor           |
| `pageSize`                                      | `number`                           | `50`    | Initial page size                 |
| `rowSelection` / `onRowSelectionChange`         | `RowSelectionState` / `OnChangeFn` | —       | Controlled row selection          |
| `expanded` / `onExpandedChange`                 | `ExpandedState` / `OnChangeFn`     | —       | Controlled expanded rows          |
| `columnVisibility` / `onColumnVisibilityChange` | `VisibilityState` / `OnChangeFn`   | —       | Controlled column visibility      |
| `enableMultiSort`                               | `boolean`                          | `false` | Allow sorting by multiple columns |
| `enableColumnReordering`                        | `boolean`                          | `false` | Drag-and-drop column reordering   |
| `enableColumnResizing`                          | `boolean`                          | `false` | Drag column borders to resize     |

### `<DataTableContent>` — Main table

Renders the full table: header, body, loading skeleton, empty state, and error state with retry.

| Prop                | Type                             | Default                 | Description                                                                                                                                       |
| ------------------- | -------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `renderExpandedRow` | `(row: Row<TData>) => ReactNode` | —                       | Content to show when a row is expanded                                                                                                            |
| `onRowClick`        | `(row: Row<TData>) => void`      | —                       | Click handler for rows                                                                                                                            |
| `emptyTitle`        | `string`                         | `"No results"`          | Title for empty state                                                                                                                             |
| `emptyDescription`  | `string`                         | `"No data to display."` | Description for empty state                                                                                                                       |
| `emptyContent`      | `ReactNode`                      | —                       | Custom empty state content                                                                                                                        |
| `stickyHeader`      | `boolean`                        | `false`                 | Pin the header while scrolling. Pair with `maxHeight`.                                                                                            |
| `maxHeight`         | `number \| string`               | —                       | Constrains the scroll area height. Required for `stickyHeader` and `virtualized`.                                                                 |
| `virtualized`       | `boolean`                        | `false`                 | Enable row virtualization. Renders only visible rows + overscan buffer. Built-in infinite scroll triggers `loadNextPage` when nearing the bottom. |
| `estimateRowHeight` | `number`                         | `48`                    | Estimated row height in px for the virtualizer                                                                                                    |
| `overscan`          | `number`                         | `5`                     | Number of rows to render outside the visible area                                                                                                 |

### `<DataTablePagination>` — Offset pagination

| Prop              | Type       | Default             | Description                       |
| ----------------- | ---------- | ------------------- | --------------------------------- |
| `pageSizeOptions` | `number[]` | `[10, 20, 50, 100]` | Page size choices                 |
| `sticky`          | `boolean`  | `false`             | Pin to bottom of scroll container |

### `<DataTableInfiniteScroll>` — Infinite scroll

In auto mode, no props needed — reads `loadNextPage` and `hasMore` from context. Uses IntersectionObserver on a sentinel element.

| Prop         | Type                          | Description                           |
| ------------ | ----------------------------- | ------------------------------------- |
| `onLoadMore` | `() => void \| Promise<void>` | Manual mode: function to load more    |
| `hasMore`    | `boolean`                     | Manual mode: whether more data exists |

### `<DataTableToolbar>` — Toolbar container

Simple flex container for composing toolbar items. Props: `className`, `children`.

### `<DataTableSearch>` — Global search

| Prop          | Type     | Default       | Description      |
| ------------- | -------- | ------------- | ---------------- |
| `placeholder` | `string` | `"Search..."` | Placeholder text |
| `debounceMs`  | `number` | `300`         | Debounce delay   |

### `<DataTableFilters>` — Dynamic column filters

Renders a "Filters" button that opens a dropdown where users dynamically add/remove column filters. Each filter type has a purpose-built inline input.

| Prop      | Type                          | Description                    |
| --------- | ----------------------------- | ------------------------------ |
| `filters` | `DataTableFilterDefinition[]` | Filter definitions (see below) |

**Filter definition types:**

```tsx
import type { DataTableFilterDefinition } from "@/components/data-table";

const filters: DataTableFilterDefinition[] = [
	{
		columnId: "role",
		title: "Role",
		type: "select",
		options: [
			{ label: "Admin", value: "admin" },
			{ label: "Editor", value: "editor" },
		],
	},
	{ columnId: "name", title: "Name", type: "text" },
	{ columnId: "score", title: "Score", type: "number" },
	{ columnId: "createdAt", title: "Created", type: "date" },
	{ columnId: "verified", title: "Verified", type: "boolean" },
];
```

**Filter types, operators, and query param encoding:**

| Type      | Operators                                           | Column filter value shape             | URL params                                                                                             |
| --------- | --------------------------------------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `select`  | (multi-select, no operator)                         | `string[]`                            | `role=admin,editor`                                                                                    |
| `text`    | `contains`, `eq`, `starts_with`, `ends_with`, `neq` | `{ op, value }`                       | `name=john&name_op=contains`                                                                           |
| `number`  | `eq`, `lt`, `gt`, `neq`, `range`                    | `{ op, value }` or `{ op, min, max }` | `score=50&score_op=gt` or `score_op=range&score_min=10&score_max=100`                                  |
| `date`    | `eq`, `before`, `after`, `neq`, `range`             | `{ op, value }` or `{ op, from, to }` | `createdAt=2024-06-01&createdAt_op=before` or `createdAt_op=range&createdAt_from=...&createdAt_to=...` |
| `boolean` | (toggle, no operator)                               | `boolean`                             | `verified=true`                                                                                        |

### `<DataTableFacetedFilter>` — Standalone column filter

For cases where you need a standalone filter button for one column (outside the dynamic filter system).

| Prop       | Type                                 | Description       |
| ---------- | ------------------------------------ | ----------------- |
| `columnId` | `string`                             | Column to filter  |
| `title`    | `string`                             | Button label      |
| `options`  | `{ label: string; value: string }[]` | Options to select |

### `<DataTableColumnToggle>` — Column visibility dropdown

| Prop    | Type     | Default     | Description  |
| ------- | -------- | ----------- | ------------ |
| `label` | `string` | `"Columns"` | Button label |

### `<DataTableExport>` — Export button

Supports current-page export (client-side) and all-data export (server-side). When `onExportAll` is provided, renders a dropdown with "Current page" and "All data" options.

| Prop          | Type                          | Default        | Description                                                         |
| ------------- | ----------------------------- | -------------- | ------------------------------------------------------------------- |
| `formatRows`  | `(rows: TData[]) => string`   | —              | Serializes current-page row data to a file string (e.g. CSV)        |
| `filename`    | `string`                      | `"export.csv"` | Filename for current-page export                                    |
| `onExportAll` | `() => void \| Promise<void>` | —              | Calls a server endpoint for full export. Use with `downloadFile()`. |
| `label`       | `string`                      | `"Export"`     | Button label                                                        |

### `<DataTableRowSelection>` — Bulk action bar

Renders only when at least one row is selected.

| Prop       | Type                                        | Description                         |
| ---------- | ------------------------------------------- | ----------------------------------- |
| `children` | `(selectedRows: Row<TData>[]) => ReactNode` | Render prop receiving selected rows |

### `<DataTableSkeleton>` — Loading skeleton

| Prop          | Type     | Default | Description                |
| ------------- | -------- | ------- | -------------------------- |
| `columnCount` | `number` | `4`     | Number of skeleton columns |
| `rowCount`    | `number` | `5`     | Number of skeleton rows    |

### `<DataTableEmpty>` — Empty state

| Prop          | Type        | Default                 | Description             |
| ------------- | ----------- | ----------------------- | ----------------------- |
| `title`       | `string`    | `"No results"`          | Empty state title       |
| `description` | `string`    | `"No data to display."` | Empty state description |
| `children`    | `ReactNode` | —                       | Custom content          |

---

## Column Reordering & Resizing

### Column reordering

Enable with `enableColumnReordering` on `<DataTable>`. Users drag columns by a grip handle to rearrange them. The new order is applied via TanStack Table's `setColumnOrder`.

```tsx
<DataTable onLoadRows={loadRows} columns={columns} enableColumnReordering>
	<DataTableContent />
	<DataTablePagination />
</DataTable>
```

**Non-draggable columns:** The select (`_select`), expand (`_expand`), and actions (`_actions`) columns are excluded from reordering. They stay pinned in their positions.

**Controlled column order:** To persist or control the order externally, use TanStack Table's `columnOrder` state:

```tsx
const [columnOrder, setColumnOrder] = useState<string[]>([]);

<DataTable
	onLoadRows={loadRows}
	columns={columns}
	enableColumnReordering
	columnOrder={columnOrder}
	onColumnOrderChange={setColumnOrder}
>
	<DataTableContent />
</DataTable>;
```

### Column resizing

Enable with `enableColumnResizing` on `<DataTable>`. Users drag the right border of column headers to resize. A blue indicator appears on hover/drag. Body cells also have resize handles for convenience.

```tsx
<DataTable onLoadRows={loadRows} columns={columns} enableColumnResizing>
	<DataTableContent />
	<DataTablePagination />
</DataTable>
```

**How it works:** On first render with data, the table measures actual DOM column widths and pre-populates TanStack's `columnSizing` state. This prevents a layout jump when the user starts resizing. Once measured, the table switches to `table-layout: fixed` with explicit widths.

**Opting out per column:** Set `enableResizing: false` on individual column definitions to prevent them from being resized. The select, expand, and actions utility columns disable resizing by default.

---

## Column Helpers & Hooks

### Column factories

```tsx
import { createSelectColumn, createExpandColumn, createActionsColumn } from "@/components/data-table";
import type { ActionDefinition } from "@/components/data-table";

const columns = [
	createSelectColumn<User>(), // Checkbox column
	createExpandColumn<User>(), // Expand toggle column
	{ accessorKey: "name", header: "Name" },
	createActionsColumn<User>((row) => [
		{ label: "Edit", onSelect: (r) => router.push(`/users/${r.id}/edit`) },
		{ label: "Copy ID", onSelect: (r) => navigator.clipboard.writeText(r.id) },
		{ type: "separator" },
		{
			label: "Delete",
			variant: "destructive",
			onSelect: async (r, { removeRow }) => {
				await api.deleteUser(r.id);
				removeRow(r.id);
			},
		},
	]),
];
```

### `createActionsColumn` — Row actions dropdown

Creates a three-dot (`⋮`) dropdown menu column. Place as the last column. The `getActions` callback receives the row data and returns action definitions.

**Action item fields:**

| Field      | Type                                                     | Default     | Description                                                                                |
| ---------- | -------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------ |
| `label`    | `string`                                                 | —           | Menu item text                                                                             |
| `onSelect` | `(row: TData, actions: DataTableActions<TData>) => void` | —           | Callback when item is clicked. Second arg provides `reloadPage`, `updateRow`, `removeRow`. |
| `variant`  | `"default" \| "destructive"`                             | `"default"` | Visual style (red for danger)                                                              |
| `disabled` | `boolean`                                                | `false`     | Disable the item                                                                           |
| `icon`     | `ReactNode`                                              | —           | Icon rendered before the label                                                             |

Use `{ type: "separator" }` between items to add a visual divider.

### Hooks

```tsx
import { useDataTable, useDataTableState, useDataTableActions } from "@/components/data-table";

// TanStack Table instance — access rows, columns, state, handlers
const table = useDataTable<User>();

// DataTable state context — loading, error, infinite scroll, feature flags
const { isLoading, hasMore, loadNextPage, error, retry, enableColumnReordering, enableColumnResizing } =
	useDataTableState();

// Data mutation actions — for optimistic updates from row actions
const { reloadPage, updateRow, removeRow } = useDataTableActions<User>();
```

### `useDataTableActions<TData>()` — Row-level data mutations

Returns typed helpers for optimistic row updates. **Only works in auto mode** (when `onLoadRows` is provided). Requires `getRowId` on `<DataTable>`.

| Method       | Signature                                                 | Description                                                                 |
| ------------ | --------------------------------------------------------- | --------------------------------------------------------------------------- |
| `reloadPage` | `() => void`                                              | Re-fetches the current page from the server with current params             |
| `updateRow`  | `(rowId: string, updater: (row: TData) => TData) => void` | Optimistically replaces a row in the current data by ID                     |
| `removeRow`  | `(rowId: string) => void`                                 | Optimistically removes a row from the current data and decrements the count |

**Usage with row actions:**

The `onSelect` callback in `createActionsColumn` receives the table actions as a second argument, giving direct access to `reloadPage`, `updateRow`, and `removeRow` without any bridge components or refs:

```tsx
import { createActionsColumn, type ActionDefinition } from "@/components/data-table";

const getRowActions = (row: User): ActionDefinition<User>[] => [
	{
		label: "Toggle status",
		onSelect: async (r, { updateRow }) => {
			const updated = await api.toggleUserStatus(r.id);
			updateRow(r.id, () => updated);
		},
	},
	{ type: "separator" },
	{
		label: "Delete",
		variant: "destructive",
		onSelect: async (r, { removeRow }) => {
			await api.deleteUser(r.id);
			removeRow(r.id);
		},
	},
];

const columns = [...baseColumns, createActionsColumn<User>(getRowActions)];

<DataTable onLoadRows={loadUsers} columns={columns} getRowId={(u) => u.id}>
	<DataTableContent />
	<DataTablePagination />
</DataTable>;
```

### Utility functions

```tsx
import { getSelectedRows, downloadFile, buildRequestSearchParams } from "@/components/data-table";

// Extract original data from selected Row objects
const users: User[] = getSelectedRows(table.getFilteredSelectedRowModel().rows);

// Trigger a browser file download from a URL
await downloadFile("/api/users/export?format=csv", "users.csv");
// Resolves filename from: explicit argument → Content-Disposition header → "export"

// Convert DataTableRequestParams to URLSearchParams
const searchParams = buildRequestSearchParams(params);
```

---

## Response Contract & Query Params

### `DataTableResponse<T>` — what `onLoadRows` must return

```typescript
// Success
{
  success: true,
  items: T[],
  page: number,        // 0-based page index
  totalPages: number,
  totalCount: number,
}

// Error — table keeps existing data, shows error state, calls onError callback
{
  success: false,
  error: { code: string, message: string },
}
```

### `DataTableRequestParams` — what `onLoadRows` receives

```typescript
{
  page: number,                   // 0-based page index
  pageSize: number,
  sorting: SortingState,          // [{ id: "columnId", desc: boolean }]
  columnFilters: ColumnFiltersState,
  globalFilter: string,
}
```

### Full query param table

`buildRequestSearchParams()` converts `DataTableRequestParams` to `URLSearchParams`:

| Param             | Type                | Example                     | Description                           |
| ----------------- | ------------------- | --------------------------- | ------------------------------------- |
| `page`            | `number`            | `0`                         | 0-based page index                    |
| `pageSize`        | `number`            | `50`                        | Items per page                        |
| `sortBy`          | `string`            | `name`                      | Column accessor to sort by            |
| `sortDesc`        | `"true" \| "false"` | `false`                     | Sort direction                        |
| `search`          | `string`            | `john`                      | Global search query                   |
| `{columnId}`      | `string`            | `role=admin,editor`         | Select filter: comma-separated values |
| `{columnId}`      | `"true" \| "false"` | `verified=true`             | Boolean filter                        |
| `{columnId}`      | `string \| number`  | `name=john`                 | Text/number/date single-value filter  |
| `{columnId}_op`   | `string`            | `name_op=contains`          | Filter operator                       |
| `{columnId}_min`  | `number`            | `score_min=10`              | Number range: minimum                 |
| `{columnId}_max`  | `number`            | `score_max=100`             | Number range: maximum                 |
| `{columnId}_from` | `string`            | `createdAt_from=2024-01-01` | Date range: start                     |
| `{columnId}_to`   | `string`            | `createdAt_to=2024-12-31`   | Date range: end                       |

---

## Building Server Endpoints

### Data Source Capability Assessment

Before wiring up a table, assess what the data source can efficiently do:

**Tier 1 — Full features** (indexed PostgreSQL via resource client):

- Server-side pagination, sorting, filtering, search, export all efficient
- Enable all table features: pagination, search, filters, column sorting, export
- Requires: proper indexes on sort/filter/search columns

**Tier 2 — Limited features** (unindexed Postgres, DynamoDB, external APIs with pagination):

- Some operations are expensive (full table scans for search, sorts on non-key columns)
- Enable: pagination and sorting on indexed/key columns only
- Warn user: "Search and filtering on [column] requires a full table scan. Consider adding an index, or use a simpler table without these features."
- DynamoDB: only sort within partition key, filter on GSI attributes

**Tier 3 — Client-side only** (Google Sheets, small datasets, APIs without pagination):

- Fetch all data, use static mode with client-side sorting/filtering
- Use `<DataTable data={allRows}>` instead of `onLoadRows`
- No server-side search/filter overhead
- Warn if dataset > ~1000 rows: "All data is loaded into the browser at once. Performance may degrade with large datasets."

The AI coder should:

1. Check resource type and schema/indexes before choosing features
2. Match table capabilities to what the data source can efficiently support
3. Proactively warn user about performance trade-offs when requested features don't match source capabilities
4. Suggest index creation (for managed Postgres) or feature reduction when appropriate

### Data Access

Data is accessed through resource clients (`@major-tech/resource-client`), NOT direct database connections. The resource client handles authentication, connection pooling, and multi-tenant isolation.

For PostgreSQL resources, the resource client executes raw SQL and returns typed results. The patterns below show the SQL to generate — pass them through the resource client's query method.

### Next.js Route Handler Pattern

All generated apps are Next.js. Server endpoints go in `app/api/.../route.ts`:

```typescript
import { NextResponse, type NextRequest } from "next/server";
import { z } from "zod";

// Zod schema for validating query params
const querySchema = z.object({
	page: z.coerce.number().int().min(0).default(0),
	pageSize: z.coerce.number().int().min(1).max(200).default(50),
	sortBy: z.string().optional(),
	sortDesc: z.enum(["true", "false"]).optional(),
	search: z.string().optional(),
});

// Whitelist of sortable columns — ALWAYS validate against this
const SORTABLE_COLUMNS = ["name", "email", "created_at", "score"] as const;

export async function GET(request: NextRequest) {
	const params = request.nextUrl.searchParams;

	try {
		const query = querySchema.parse(Object.fromEntries(params));

		// 1. Build WHERE clause from filters
		const { whereClause, values } = buildWhereClause(params);

		// 2. Validate sort column against whitelist
		const sortColumn = SORTABLE_COLUMNS.includes(query.sortBy as any) ? query.sortBy : "created_at";
		const sortDir = query.sortDesc === "true" ? "DESC" : "ASC";

		// 3. Run count + data queries (can be parallel for better latency)
		const offset = query.page * query.pageSize;

		const [countResult, dataResult] = await Promise.all([
			resourceClient.invoke(`SELECT COUNT(*) as count FROM users ${whereClause}`, values, "count-users"),
			resourceClient.invoke(
				`SELECT * FROM users ${whereClause} ORDER BY ${sortColumn} ${sortDir} LIMIT $${values.length + 1} OFFSET $${values.length + 2}`,
				[...values, query.pageSize, offset],
				"list-users",
			),
		]);

		if (!countResult.ok || !dataResult.ok) {
			throw new Error("Database query failed");
		}

		const totalCount = Number(countResult.result.rows[0].count);

		return NextResponse.json({
			success: true,
			items: dataResult.result.rows,
			page: query.page,
			totalPages: Math.ceil(totalCount / query.pageSize),
			totalCount,
		});
	} catch (error) {
		console.error("Failed to fetch data:", error);
		return NextResponse.json(
			{ success: false, error: { code: "QUERY_FAILED", message: "Failed to fetch data" } },
			{ status: 500 },
		);
	}
}
```

### Building Postgres WHERE Clauses

Translate the table's query params into parameterized SQL. Every filter type maps to specific SQL:

```typescript
interface WhereResult {
	whereClause: string;
	values: (string | number | boolean)[];
}

function buildWhereClause(params: URLSearchParams): WhereResult {
	const conditions: string[] = [];
	const values: (string | number | boolean)[] = [];
	let paramIndex = 1;

	// Global search — OR across multiple columns
	const search = params.get("search");
	if (search) {
		const q = `%${search}%`;
		conditions.push(`(name ILIKE $${paramIndex} OR email ILIKE $${paramIndex + 1})`);
		values.push(q, q);
		paramIndex += 2;
	}

	// Select filter: comma-separated values → IN clause
	const role = params.get("role");
	if (role) {
		const roles = role.split(",");
		const placeholders = roles.map((_, i) => `$${paramIndex + i}`).join(", ");
		conditions.push(`role IN (${placeholders})`);
		values.push(...roles);
		paramIndex += roles.length;
	}

	// Text filter with operator
	const name = params.get("name");
	if (name) {
		const op = params.get("name_op") ?? "contains";
		switch (op) {
			case "contains":
				conditions.push(`name ILIKE $${paramIndex}`);
				values.push(`%${name}%`);
				break;
			case "eq":
				conditions.push(`LOWER(name) = LOWER($${paramIndex})`);
				values.push(name);
				break;
			case "starts_with":
				conditions.push(`name ILIKE $${paramIndex}`);
				values.push(`${name}%`);
				break;
			case "ends_with":
				conditions.push(`name ILIKE $${paramIndex}`);
				values.push(`%${name}`);
				break;
			case "neq":
				conditions.push(`LOWER(name) != LOWER($${paramIndex})`);
				values.push(name);
				break;
		}
		paramIndex++;
	}

	// Number filter with operator
	const scoreOp = params.get("score_op") ?? "eq";
	if (scoreOp === "range") {
		const min = params.get("score_min");
		const max = params.get("score_max");
		if (min) {
			conditions.push(`score >= $${paramIndex}`);
			values.push(Number(min));
			paramIndex++;
		}
		if (max) {
			conditions.push(`score <= $${paramIndex}`);
			values.push(Number(max));
			paramIndex++;
		}
	} else {
		const score = params.get("score");
		if (score) {
			const num = Number(score);
			switch (scoreOp) {
				case "eq":
					conditions.push(`score = $${paramIndex}`);
					break;
				case "lt":
					conditions.push(`score < $${paramIndex}`);
					break;
				case "gt":
					conditions.push(`score > $${paramIndex}`);
					break;
				case "neq":
					conditions.push(`score != $${paramIndex}`);
					break;
			}
			values.push(num);
			paramIndex++;
		}
	}

	// Date filter with operator
	const dateOp = params.get("created_at_op") ?? "eq";
	if (dateOp === "range") {
		const from = params.get("created_at_from");
		const to = params.get("created_at_to");
		if (from) {
			conditions.push(`created_at >= $${paramIndex}`);
			values.push(from);
			paramIndex++;
		}
		if (to) {
			conditions.push(`created_at <= $${paramIndex}`);
			values.push(to);
			paramIndex++;
		}
	} else {
		const dateVal = params.get("created_at");
		if (dateVal) {
			switch (dateOp) {
				case "eq":
					conditions.push(`created_at::date = $${paramIndex}::date`);
					break;
				case "before":
					conditions.push(`created_at < $${paramIndex}`);
					break;
				case "after":
					conditions.push(`created_at > $${paramIndex}`);
					break;
				case "neq":
					conditions.push(`created_at::date != $${paramIndex}::date`);
					break;
			}
			values.push(dateVal);
			paramIndex++;
		}
	}

	// Boolean filter
	const verified = params.get("verified");
	if (verified) {
		conditions.push(`verified = $${paramIndex}`);
		values.push(verified === "true");
		paramIndex++;
	}

	const whereClause = conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "";
	return { whereClause, values };
}
```

### Sorting with Whitelist Validation

Never interpolate user input directly into ORDER BY. Always validate against a whitelist:

```typescript
const SORTABLE_COLUMNS: Record<string, string> = {
	name: "name",
	email: "email",
	created_at: "created_at",
	score: "score",
};

function buildOrderBy(params: URLSearchParams): string {
	const sortBy = params.get("sortBy");
	const sortDesc = params.get("sortDesc") === "true";

	const column = sortBy && SORTABLE_COLUMNS[sortBy];
	if (!column) {
		return "ORDER BY created_at DESC"; // default sort
	}

	return `ORDER BY ${column} ${sortDesc ? "DESC" : "ASC"}`;
}
```

### Indexing Best Practices

Create indexes that match your table's common query patterns:

```sql
-- Composite index for common sort + filter combos
CREATE INDEX idx_users_role_created ON users (role, created_at DESC);

-- Partial index for filtered subsets (only rows matching condition are indexed)
CREATE INDEX idx_users_active ON users (created_at DESC) WHERE status = 'active';

-- Expression index for case-insensitive text search
CREATE INDEX idx_users_name_lower ON users (LOWER(name));

-- GIN index for trigram similarity (ILIKE '%term%' acceleration)
-- Requires: CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_users_name_trgm ON users USING GIN (name gin_trgm_ops);

-- GIN index for full-text search
CREATE INDEX idx_users_search ON users USING GIN (to_tsvector('english', name || ' ' || email));
```

**When to add indexes:**

- Add indexes on columns that appear in WHERE, ORDER BY, or JOIN clauses
- Composite indexes: put equality conditions first, range/sort columns last
- Partial indexes: use when queries consistently filter to a subset (e.g. `WHERE deleted_at IS NULL`)
- Skip indexes on low-cardinality columns (e.g. boolean with 50/50 split) — the planner won't use them
- Trade-off: indexes speed up reads but slow down writes. For write-heavy tables with infrequent reads, keep indexing minimal

### Text Search Strategies

No Elasticsearch — apps use SQL/NoSQL via resource connectors. Choose the strategy based on dataset size:

**Small datasets (<10k rows) — plain ILIKE:**

```sql
-- No special setup needed. Fast enough for small tables.
SELECT * FROM users
WHERE name ILIKE '%search%' OR email ILIKE '%search%';
```

**Medium datasets (10k–500k rows) — pg_trgm + GIN index:**

```sql
-- Setup (one-time): enable extension and create index
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_users_name_trgm ON users USING GIN (name gin_trgm_ops);

-- Query: same ILIKE, but now index-accelerated
SELECT * FROM users
WHERE name ILIKE '%search%' OR email ILIKE '%search%';
```

**Large datasets (500k+ rows) — tsvector full-text search:**

```sql
-- Setup: add a stored generated column + GIN index
ALTER TABLE users ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (to_tsvector('english', coalesce(name, '') || ' ' || coalesce(email, ''))) STORED;
CREATE INDEX idx_users_fts ON users USING GIN (search_vector);

-- Query: use plainto_tsquery for user input
SELECT * FROM users
WHERE search_vector @@ plainto_tsquery('english', 'search term')
ORDER BY ts_rank(search_vector, plainto_tsquery('english', 'search term')) DESC;
```

**Non-Postgres sources** (Google Sheets, DynamoDB, external APIs):

- Google Sheets: fetch all rows, filter client-side in static mode
- DynamoDB: use FilterExpression on scan (slow) or design GSIs for searchable attributes
- External APIs: delegate to API's own search params if available, else fetch and filter client-side

### Postgres Pagination Techniques

**OFFSET/LIMIT — use with `<DataTablePagination>`:**

The standard approach. Works with the table's 0-based page index.

```sql
SELECT * FROM users
WHERE ...
ORDER BY created_at DESC
LIMIT 50 OFFSET 100;  -- page 2, pageSize 50
```

Trade-offs:

- Simple and works with arbitrary page jumps
- Performance degrades on deep pages (Postgres must scan and discard `OFFSET` rows)
- Fine for most tables (users rarely page past page 10–20)
- If performance matters at depth 1000+, consider keyset pagination

**Keyset/cursor pagination — use with `<DataTableInfiniteScroll>`:**

For infinite scroll on very large datasets. Uses the last row's sort key as the cursor.

```sql
-- First page
SELECT * FROM users ORDER BY created_at DESC, id DESC LIMIT 50;

-- Next page (cursor = last row's created_at + id)
SELECT * FROM users
WHERE (created_at, id) < ($1, $2)
ORDER BY created_at DESC, id DESC
LIMIT 50;
```

Trade-offs:

- Consistent performance regardless of depth
- No arbitrary page jumps (forward-only)
- Sort key must be unique or combined with a tiebreaker (e.g. `id`)
- Great for infinite scroll; not suitable for offset pagination

**Efficient COUNT:**

For offset pagination, you need `totalCount` for the response:

```sql
-- Exact count — run in parallel with data query
SELECT COUNT(*) FROM users WHERE ...;

-- For very large tables (10M+), consider an approximate count when no filters applied:
SELECT reltuples::bigint AS estimate FROM pg_class WHERE relname = 'users';
-- This is a catalog estimate, updated by ANALYZE. Only use when precision isn't critical.
```

Best practice: run the COUNT and data query in parallel using `Promise.all` (shown in the route handler pattern above). This halves the perceived latency.

### Export Route Handler

Streaming CSV export endpoint. Applies the same filters/sort as the table but without pagination.

```typescript
import { NextResponse, type NextRequest } from "next/server";

export async function GET(request: NextRequest) {
	const params = request.nextUrl.searchParams;

	try {
		const { whereClause, values } = buildWhereClause(params);
		const orderBy = buildOrderBy(params);

		// No LIMIT/OFFSET — fetch all matching rows
		const result = await resourceClient.invoke(`SELECT * FROM users ${whereClause} ${orderBy}`, values, "export-users");

		if (!result.ok) {
			throw new Error("Database query failed");
		}

		// Build CSV
		const headers = ["ID", "Name", "Email", "Role", "Created At"];
		const csvRows = [
			headers.join(","),
			...result.result.rows.map((row) =>
				[row.id, row.name, row.email, row.role, row.created_at]
					.map((v) => `"${String(v ?? "").replace(/"/g, '""')}"`)
					.join(","),
			),
		];
		const csv = csvRows.join("\n");

		return new NextResponse(csv, {
			headers: {
				"Content-Type": "text/csv",
				"Content-Disposition": 'attachment; filename="users-export.csv"',
			},
		});
	} catch (error) {
		console.error("Export failed:", error);
		return NextResponse.json(
			{ success: false, error: { code: "EXPORT_FAILED", message: "Export failed" } },
			{ status: 500 },
		);
	}
}
```

Frontend usage with `<DataTableExport>`:

```tsx
import { DataTableExport, downloadFile } from "@/components/data-table";

<DataTableExport
	formatRows={(rows) => {
		const headers = ["Name", "Email", "Role"];
		return [headers.join(","), ...rows.map((r) => [r.name, r.email, r.role].map((v) => `"${v}"`).join(","))].join("\n");
	}}
	filename="users.csv"
	onExportAll={async () => {
		// Calls the export route with current filters applied
		const params = buildRequestSearchParams(currentParams);
		await downloadFile(`/api/users/export?${params}`);
	}}
/>;
```

---

## Re-exported Primitives

For custom table layouts that don't use `<DataTableContent>`:

```tsx
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
	Button,
	Input,
	Checkbox,
	Select,
	SelectValue,
	SelectTrigger,
	SelectContent,
	SelectItem,
	DropdownMenu,
	DropdownMenuTrigger,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
} from "@/components/data-table";
```
