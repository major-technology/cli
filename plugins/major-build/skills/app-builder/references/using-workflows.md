# Starting Workflows from an App

Generate the client with `sandbox_add-workflow-client` and call it from backend code:

- `run({ input })` starts a run and returns `{ runId }`.
- `getRun({ runId })` returns `{ run, document }`: poll `run.status`; the output is `document.output`.
- A `human_approval` with an `app` channel POSTs `{ runId, executionId, message, payload, options }` to the app's `path`. Store it, then answer from the user's click with `approvals.respond({ runId, executionId, choice })`. The first answer wins.
