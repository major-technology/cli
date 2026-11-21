# Starting Workflows from an App

Generate the client with `sandbox_add-workflow-client` and use the import it returns, from backend code only.

```typescript
import { leadEnrichmentWorkflowClient } from "./clients"; // the import add-workflow-client returned

const { runId } = await leadEnrichmentWorkflowClient.run({ input: { leadId } });
const { run, document } = await leadEnrichmentWorkflowClient.getRun({ runId }); // poll until run.status !== "running"; output in document?.output
```

A `human_approval` with an `app` channel POSTs an `AppApprovalRequest` (`{ runId, executionId, message, payload, options }`) to the app's `path`. Store it, show the options, and answer from the user's click:

```typescript
await leadEnrichmentWorkflowClient.approvals.respond({ runId, executionId, choice: "approve" });
```

The first answer wins; a later one gets a 409.
