# Caching in a Major app

Pick the cache from the staleness the user agreed to (see the table in `SKILL.md`). The examples use a PostgreSQL client named `ordersDbClient`; read the generated client in `clients/` for the real name and method signatures.

## Same call repeated within one request: React `cache()`

When a layout, a page and a component each need the same data during one render, wrap the loader so it runs once. Nothing is kept between requests, so nothing can go stale.

```typescript
import "server-only";
import { cache } from "react";
import { ordersDbClient } from "@/clients";

export const getAccount = cache(async (accountId: string) => {
	const result = await ordersDbClient.invoke<{ id: string; name: string }>(
		"SELECT id, name FROM accounts WHERE id = $1",
		[accountId],
		"get-account",
	);

	if (!result.ok) {
		throw new Error("Failed to load account");
	}

	return result.result.rows[0];
});
```

## Seconds to an hour: in-process cache

An app runs as one process, so a cache held in memory is seen by every request. Keep it on `globalThis`: Next.js can load a module more than once, and a plain module variable would then give each copy its own cache.

```typescript
// lib/ttl-cache.ts
import "server-only";

type Entry = { expiresAt: number; value: Promise<unknown> };

const MAX_ENTRIES = 200;
const store = ((globalThis as { __ttlCache?: Map<string, Entry> }).__ttlCache ??= new Map());

export function cached<T>(key: string, ttlSeconds: number, load: () => Promise<T>): Promise<T> {
	const hit = store.get(key);

	if (hit && hit.expiresAt > Date.now()) {
		return hit.value as Promise<T>;
	}

	// Store the promise, not the result, so requests arriving together share one load.
	const value = load().catch((error) => {
		store.delete(key);
		throw error;
	});

	store.delete(key);
	store.set(key, { expiresAt: Date.now() + ttlSeconds * 1000, value });

	if (store.size > MAX_ENTRIES) {
		store.delete(store.keys().next().value!);
	}

	return value;
}

export function invalidate(key: string): void {
	store.delete(key);
}
```

```typescript
const summary = await cached(`sales-summary:${region}`, 300, () => loadSalesSummary(region));
```

Rules:

- **Put everything the result depends on into the key**: the parameters, and the user or role id whenever the data differs by person.
- **Call `invalidate(key)` after a write** when the user must see their own change immediately.
- **Keep entries small.** The cache lives inside the app's memory limit (128 MiB by default). Cache summaries and single pages of rows, never whole tables.
- **It empties on every deploy and restart**, and when a scale-to-zero app goes to sleep. The first request afterwards pays the full cost, so this cache suits loads of a few seconds at most.

## An hour or more, or a source that takes many seconds: snapshot table

Copy the slow result into a table in the app's managed database on a schedule, and have pages read the table. Page loads then cost one fast query however slow the source is, and the data survives deploys. This is the right cache for warehouses (Snowflake, BigQuery) and slow third-party APIs.

1. Create the table with `mcp__plugin_major_major__run_migration`. Set up the managed database first if the app has none (see `using-connectors` → managed-database). Give the table a `refreshed_at` column.
2. Add a route that reloads it, for example `app/api/snapshots/sales-summary/route.ts`: query the source, then replace the rows.
3. Schedule it: load `workflow-builder` and build a workflow with a cron trigger and an `app_call` node that calls the route. Set the schedule to the staleness the user agreed to.
4. Read the table from the page and show `refreshed_at` as "Updated 12 minutes ago".

```typescript
// app/api/snapshots/sales-summary/route.ts
import { warehouseClient, appDbClient } from "@/clients";

export async function POST() {
	const source = await warehouseClient.invoke<{ region: string; revenue: number }>(
		"SELECT region, SUM(amount) AS revenue FROM orders GROUP BY region",
		[],
		"snapshot-sales-summary-read",
	);

	if (!source.ok) {
		return Response.json({ error: "Source query failed" }, { status: 502 });
	}

	// One statement, so readers never see an empty or half-filled table.
	const written = await appDbClient.invoke(
		`WITH incoming AS (
			SELECT * FROM jsonb_to_recordset($1::jsonb) AS r(region text, revenue numeric)
		), removed AS (
			DELETE FROM sales_summary WHERE region NOT IN (SELECT region FROM incoming)
		)
		INSERT INTO sales_summary (region, revenue, refreshed_at)
		SELECT region, revenue, now() FROM incoming
		ON CONFLICT (region) DO UPDATE SET revenue = EXCLUDED.revenue, refreshed_at = EXCLUDED.refreshed_at`,
		[JSON.stringify(source.result.rows)],
		"snapshot-sales-summary-write",
	);

	if (!written.ok) {
		return Response.json({ error: "Snapshot write failed" }, { status: 500 });
	}

	return Response.json({ rows: source.result.rows.length });
}
```

Notes:

- Keep the failed-refresh behavior safe: when the source query fails, leave the old rows in place and return an error so the workflow run shows it.
- Run the route once after creating it so the first visitor does not see an empty table.
- If the user also wants a manual refresh, add a "Refresh" button that calls the same route.
- When the slow query runs on the managed database itself, a materialized view does the same job with less code: create it with `run_migration`, give it a unique index, and have the scheduled route run `REFRESH MATERIALIZED VIEW CONCURRENTLY <name>`.

## The user's own browser: `Cache-Control`

For a GET route the browser fetches repeatedly (reference lists, filter options), let the browser reuse its copy. Always use `private`: the response was authorized for one user.

```typescript
return Response.json(options, {
	headers: { "Cache-Control": "private, max-age=60" },
});
```

This saves repeat requests from one person. It does nothing for other users or for server-rendered pages.

## What not to use

`unstable_cache`, `"use cache"` and a page-level `export const revalidate` do not work around resource calls. The generated clients read the incoming request's headers on every call so that Major can authorize it. Next.js does not allow that inside a cached function, and any page that makes a resource call is rendered per request regardless of `revalidate`.
