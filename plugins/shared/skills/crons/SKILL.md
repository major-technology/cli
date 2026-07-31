---
name: using-crons
description: Use when the user needs to run something on a cadence or do anything that interacts with crons or scheduled jobs. Scheduled work runs through Major workflows — a cron trigger calling a route on the deployed app.
---

# Scheduled Jobs (Workflows)

Scheduled work on Major is handled by **workflows**: graphs of steps run by the platform's workflow engine. A scheduled job is a workflow with a **cron trigger** and an **`app_call` step** that calls a route on the deployed app on a cadence.

The legacy `cron.json` system is retired — the file is no longer read on deploy, and existing crons were migrated to workflows automatically. If the project still has a `cron.json`, it has no effect and can be deleted.

## Building a scheduled job

Both halves are yours to build:

1. **The route** — implement an HTTP route in the app that performs the work (see below), and deploy it: workflows call the deployed app.
2. **The workflow** — invoke the **`workflow-builder`** skill and follow it to build the workflow itself: create the workflow, define a cron trigger and an `app_call` node targeting the route's method and path, then save, test, and publish through that skill's lifecycle.

## The route

```typescript
// app/api/jobs/cleanup/route.ts
export async function POST() {
	// ... remove expired sessions
	return Response.json({ ok: true });
}
```

How workflow calls reach the route:

- Calls go to the **deployed** app, so the app must be deployed before the workflow can call the route.
- Requests arrive authenticated as the Major user the workflow runs as (via a platform-signed app-call JWT) — the route needs no webhook access and the app does not need to be public.
- The step's configured input arrives as query params for GET/DELETE and as a JSON body for POST/PUT/PATCH.
- The response body becomes the step's output, available to later steps in the workflow.

## Cron schedules

Cron triggers use a 5-field cron expression (`minute hour day-of-month month day-of-week` — no seconds field) with an IANA timezone. Common schedules:

| Schedule               | Expression    |
| ---------------------- | ------------- |
| Every 5 minutes        | `*/5 * * * *` |
| Every hour             | `0 * * * *`   |
| Every day at 2 AM      | `0 2 * * *`   |
| Every Monday 9 AM      | `0 9 * * 1`   |
| 1st of month, midnight | `0 0 1 * *`   |
