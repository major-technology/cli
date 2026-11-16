---
name: app-performance
description: Use this skill when the user says a Major app is slow, laggy, sluggish, takes a long time to load, hangs, spins, or times out, or asks to speed up, optimize, tune, cache, or scale an app, page, query, or API route. Covers measuring where the time goes (resource-call timings, CPU and memory usage), fixing slow and chatty queries, choosing a cache and agreeing its staleness with the user, and resizing the app's CPU and memory. Do not use it for failures that are not about speed — that is debug-issue.
---

# Making a Major app faster

Measure, fix the largest cost, measure again. Never tune from a reading of the code alone: in production the time is almost always in a few resource calls, and the numbers say which.

Work in the app's mounted sandbox (load `app-builder` to mount it). Below, `sandbox_*` means `mcp__plugin_major_major__sandbox_*`, and `major` commands run through `sandbox_bash` in `/workspace/app`.

## How a request is served

These facts decide which fixes work:

- **One small pod per app.** Every deployed app runs as exactly one Node process. A new app gets 0.1 CPU and 128 MiB of memory. Autoscaling doubles a limit after CPU stays above 85% of it for five minutes, or after an out-of-memory kill, so short bursts never trigger it.
- **Every browser request is authenticated first**, including each static asset, and nothing is served from a CDN. Fewer and smaller requests matter more here than on a typical host.
- **Every resource call is a network round trip** from the app out to Major, which authorizes it, runs it against the data source, and records it. That fixed cost is paid per call on top of the query itself, so 200 fast queries are slower than one query returning 200 rows.
- **Major opens at most 8 connections per database.** More than 8 concurrent queries to one database wait in line.
- **PostgreSQL queries time out at 10 seconds** unless the call or the resource sets another limit.
- **There is no per-route latency data.** Timings exist per resource operation, not per page or API route.

## 1. Measure

Ask which page or action is slow if the user has not said. Then collect all three:

**Resource operations.** `major resource invocations --days 7` returns every operation the deployed app ran, sorted by total time: `operationKey`, `calls`, `p50Ms`, `p95Ms`, `totalMs`, `errorRate`, `resourceSubtype`, and `callSite` (`filePath`, `line`, `sourceText`). Add `--slow` to keep only operations over 500 ms p95, and `--environment coding-session` for preview traffic. `p50Ms` and `p95Ms` cover the data source only, not the round trip around it.

**Compute.** `mcp__plugin_major_major__get_app_scaling` with the `applicationId` returns the current CPU and memory limits, whether autoscaling and scale-to-zero are on, recent scaling events (`cpu_high`, `oom`), and usage against the limits.

**The slow route itself.** Time it in the preview: `curl -s -o /dev/null -w '%{time_total}\n' http://localhost:3000/<path>`. Run it three times and ignore the first, which includes compilation. The preview runs in development mode on a much larger machine, so compare runs with each other, never with production. To see where a route spends its time, wrap its steps in `console.time` / `console.timeEnd`, read them with `major app logs --preview`, and remove them before committing.

## 2. Diagnose

Match what you measured to one of these. Rank by `totalMs`: an operation the user never complained about can still be what saturates the app.

| What you see | Cause | Fix |
| --- | --- | --- |
| High `calls`, low `p50Ms`, high `totalMs` | A query inside a loop, or a poll | Batch it; slow or merge the poll |
| `p95Ms` near 10,000 (or another round number) with a non-zero `errorRate` | The query hits its timeout | Fix the query; do not just raise `timeoutMs` |
| SQL operation with high `p50Ms` | Missing index, or too many rows read | EXPLAIN, then index or narrow |
| Several operations with the same `calls`, each slow | One page running a group of queries | Run them in parallel, then cache |
| Warehouse (Snowflake, BigQuery) or third-party API with high `p50Ms` | The source is slow by nature | Cache or snapshot; nothing in the query will fix it |
| `p95Ms` many times `p50Ms` | Depends on the parameters, or a cold source | Find the slow parameter values in `major app logs` |
| Route is slow but its operations are fast | Calls awaited one after another, CPU-heavy work, or a huge payload | Parallelize, move work into SQL, send less |
| CPU at its limit, or `cpu_high` / `oom` events | The pod is too small | Resize (step 5) |
| First request after a quiet period is slow, later ones fast | Scale-to-zero is on and the app was asleep | Turn it off (step 5) |

## 3. Fix the code

**SQL.** Run the statement from `callSite.sourceText` through `mcp__resources__postgresql_query` with `EXPLAIN (ANALYZE, BUFFERS)` and realistic parameters. Only do this for a `SELECT`: `EXPLAIN ANALYZE` executes the statement. Look for sequential scans on large tables and for row estimates far from the actual counts.

- Add the index the plan is missing. On a managed database use `mcp__plugin_major_major__run_migration` with `CREATE INDEX CONCURRENTLY`. On an external database, give the user the statement and let its owner run it; never change an external schema yourself.
- Select the columns the page uses, not `SELECT *`.
- Filter, sort, aggregate and paginate in SQL. Never fetch a table to reduce it in JavaScript: on 0.1 CPU, mapping 50,000 rows is itself slow.
- Paginate on the server. For deep pages use keyset pagination (`WHERE id > $1 ORDER BY id LIMIT $2`), not `OFFSET`.

**Chatty calls.** Replace a query per item with one query: `WHERE id = ANY($1)`, or a `JOIN`. For an API without a batch endpoint, fetch once and index the result by id in memory.

**Sequential calls.** Start independent calls together with `Promise.all`. Keep at most 8 in flight to one database.

**Polling.** A count polled every few seconds by every open tab adds up to tens of thousands of calls a day. Lengthen the interval, stop polling while the tab is hidden, and return everything one screen polls from a single route.

**Rendering.**

- Fetch in Server Components and wrap slow sections in `<Suspense>`, so the page shell shows at once and slow data streams in.
- Do not start a fetch from a `useEffect` that waits on another fetch. Load the data on the server in one pass.
- Load heavy client-only libraries (charts, editors, maps) with `next/dynamic`.
- Send the browser only what it renders. A table holds one page of rows, not the whole result.

**Long work.** Anything that takes more than a few seconds for good reason (exports, syncs, AI calls) should not block a page load. Move it into a workflow (load `workflow-builder`) and show its status.

## 4. Cache

Cache when the source is slow by nature or the same result is read far more often than it changes. Fix a bad query first; a cache over a missing index only hides it until the cache expires.

**Agree the staleness with the user before writing any cache.** Use `AskUserQuestion`, one question per data set, naming the screen and the data:

- How old may this be: always live, about a minute, 15 minutes, an hour, a day?
- When someone changes it in the app, must they see their change immediately?
- Is it the same for everyone, or does it differ by user, role or team?
- Would a "Refresh" button with a "last updated" time be acceptable?

Do not pick a lifetime yourself, and do not cache data whose answers you do not have. Data that differs by user must be keyed by that user or left uncached: a shared cache hands one person's rows to the next.

Then choose the cache by lifetime. [references/caching.md](references/caching.md) has the code for each.

| Acceptable staleness | Cache |
| --- | --- |
| None, but the same call repeats within one request | React `cache()` |
| Seconds to an hour | In-process cache with a time limit |
| An hour or more, or the source takes many seconds | Snapshot table refreshed on a schedule |
| The user's own browser, a few seconds to minutes | `Cache-Control: private` on a GET route |

Next.js's own data caches (`unstable_cache`, `"use cache"`, a page-level `revalidate`) do not work around resource calls: the generated clients read the request headers on every call, which Next.js forbids inside a cached function and which makes every such page dynamic. Use the caches above instead.

Whenever cached data can be more than a minute old, show its age on the screen.

## 5. Resize the app

More CPU or memory helps only when the pod itself is the limit. It does nothing for a slow query: that time is spent waiting on the data source.

Resize when `get_app_scaling` shows CPU regularly near its limit, `cpu_high` or `oom` events, or when the work is truly CPU-bound (rendering large pages, generating files, parsing big payloads) and step 3 has already moved what it could into SQL.

`mcp__plugin_major_major__set_app_scaling` takes the `applicationId` and any of `cpuLimit`, `memoryLimit`, `autoscalingEnabled`, `scaleToZero`.

- Apps are billed on their limits. Tell the user the current and new monthly estimate (both `get_app_scaling` and the result of a change report it) and get their go-ahead before calling it.
- A change redeploys the app.
- Go up one step at a time (double the limit), then measure again. Limits cannot go below the default or above the cap that `get_app_scaling` reports.
- Leave autoscaling on unless the user wants a fixed cost.
- Turning `scaleToZero` off keeps the app always running, which removes the cold start and costs more.

## 6. Verify and report

Repeat the measurement that showed the problem. Time the route again in the preview, exercise it a few times, and compare `major resource invocations --environment coding-session --days 1` with the earlier numbers. Lint, commit and push as `app-builder` describes. Production numbers only move after a deploy and some real traffic; say so instead of claiming a result you have not seen.

Report to the user:

- What was slow, with the numbers before.
- What you changed, and the numbers after.
- Each cache you added: what it holds, how stale it can be, and how it is refreshed.
- Any resize, with its cost.
- What is still slow and what it would take to fix.
