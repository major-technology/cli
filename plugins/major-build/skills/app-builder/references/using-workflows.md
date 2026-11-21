# Starting Workflows from an App

A Major app can start a **workflow** run from its backend, read the run back, and answer a `human_approval` the workflow hands to the app.

## Generated clients — never hand-write them

Call `sandbox_add-workflow-client` (find the id with the `list` tool, `target_type: "workflow"`). It generates a typed singleton into `backend/src/clients/` (`clients/` in a Next.js app) and returns the import line — **use that import VERBATIM**. The client is already bound to the workflow, and deploying grants the app permission to start exactly the workflows it has clients for.

To stop using a workflow, delete the code that references it and call `sandbox_remove-workflow-client`.

Backend only, like every Major client: the frontend calls your own `/api` route, which calls the workflow. No config is needed; the platform injects the credentials while building and after deployment.

## Methods

```typescript
import { leadEnrichmentWorkflowClient } from "./clients"; // use the import add-workflow-client returned, verbatim

// Start a run of the published version. `input` must match the workflow's input_schema.
// Returns as soon as the run is accepted; the workflow runs async.
const { runId } = await leadEnrichmentWorkflowClient.run({ input: { leadId } });

// Read back a run this app started. Poll until run.status is no longer "running".
const { run, document } = await leadEnrichmentWorkflowClient.getRun({ runId });
// run.status: "running" | "completed" | "failed" | "canceled"
// completed => document?.output is the end node's output; failed or canceled => run.error
```

The app's deployer (the preview's owner while building) must be allowed to use the workflow, or `run` is refused (403). The run executes as the workflow's publisher, like any published run, and records the person who clicked as its invoker. A run started from a preview calls back into that preview.

## Answering an approval the workflow hands to the app

A `human_approval` node with an `app` channel (`{"type": "app", "app_id": ..., "path": ...}`) POSTs this to the app at `path`, then waits:

```typescript
import type { AppApprovalRequest } from "@major-tech/workflows-client";
// { runId, executionId, workflowId, message, payload, options: [{ value, label?, style? }] }
```

- Store it keyed by `(runId, executionId)` and return a 2xx. The 2xx only acknowledges delivery; a delivery retry resends the same pair, so ignore a duplicate.
- Show `message`, `payload` and the `options` in the app, and let the user pick one.
- Answer with the workflow's client, from the route the user's click hits (it carries their identity, recorded as `responded_by`):

```typescript
await leadEnrichmentWorkflowClient.approvals.respond({ runId, executionId, choice: "approve" }); // choice: one of options[].value
```

The first answer wins: answering again, or after `on_timeout` resolved it, is a conflict (HTTP 409), so show it as already decided. Any user of the app can answer; check who may in your own route.
