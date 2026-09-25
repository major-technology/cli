# Triggering Agents from an App

A Major app can trigger **agents** at runtime and interact with their runs. The canonical use case is a "daily command center" whose page load kicks off several agents (summarize my email, prep my calendar, review my PRs) that populate the UI and suggest actions.

The platform injects the same credentials while building and after deployment, so you can test the full trigger -> run -> read loop before deploying. Runs started while building route the agent's callbacks back to the app version you are working on.

A **run is a chat thread.** The `chatThreadId` returned by `run()` is the `runId` you pass to every run-op (`sendMessage` / `stopAgent` / `getAgentContent`). Runs are asynchronous (the agent executes in the background), so `run()` returns immediately — poll `getAgentContent` / `getRunningInstancesOfAgent` for progress.

## Generated clients — never hand-write them

Always call `sandbox_add-agent-client` first (find the id with the `list` tool, `target_type: "agent"`). It generates a typed singleton into `clients/` and returns the import line — **use that import VERBATIM**. Do not `createAgentsClient()`, do not `getAgentId()`, do not bake a uuid. The generated client already binds the agent id.

To stop triggering an agent, delete the code that references it and call `sandbox_remove-agent-client`.

## If the agent should post back into the app

Triggering an agent does not let it call the app. If the agent should post results, write data, or hit an endpoint on this app, grant it application access by adding the app's id to the `applications` array in the agent's `agent.jsonc`, then push and publish the agent (the agent-builder skill covers the flow). Without that, the agent's requests to the app are rejected (HTTP 403).

## Setup

```typescript
import { supportBotClient } from "./clients"; // use the import add-agent-client returned, verbatim
```

- **No config needed.** `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are injected by the platform while building and after deployment; the generated client reads them.
- **Server-side only** (Next.js): use the client in Server Components, Server Actions, Route Handlers — never in client components (it sends the deployment-identity token).

## Methods

Read the package's types before calling — these are the exact signatures. `run` does **not** take `agentId`; the generated client is already bound to one agent.

```typescript
import { supportBotClient } from "./clients"; // use the import add-agent-client returned, verbatim

// Start a run. Returns as soon as the run is accepted; the agent runs async.
const { chatThreadId } = await supportBotClient.run({
	prompt: "Summarize today's unread email and flag what needs a reply.",
	name: "Daily email summary", // optional, shown in the Major UI
});
// chatThreadId is the runId for every run-op below.

// Send a follow-up message to a run this app started.
await supportBotClient.sendMessage(chatThreadId, "Now draft replies for the urgent ones.");

// Stop a run. Idempotent — stopping an already-finished run still succeeds.
await supportBotClient.stopAgent(chatThreadId);

// List the runs this app started that are still executing for this agent.
const running = await supportBotClient.getRunningInstancesOfAgent();
// → AgentRun[]: { runId, agentId, status: "running", startedAt }

// Read the most recent messages of a run's thread (n caps how many).
const messages = await supportBotClient.getAgentContent(chatThreadId, 20);
// → AgentMessage[]: { role, type, content, timestamp }
```

## Approving tool calls a run is paused on

A run can pause mid-execution when the agent wants to call a **permission-gated tool** (e.g. a connector action or an endpoint on your app). The run blocks until someone approves or denies the call. If your app surfaces these, the user can act on them without leaving your UI; left unanswered, each approval auto-denies at its `expiresAt` and the run continues as a denial.

```typescript
// List the tool calls this run is currently paused on. Empty array when none.
// Poll it (on thread open, or from a cron) while the run is live.
const pending = await supportBotClient.listPendingApprovals(chatThreadId);
// → PendingApproval[]: { approvalId, toolName, toolArgs, description?, expiresAt? }

// Approve (or deny) one, unblocking the run.
await supportBotClient.respondToApproval(chatThreadId, pending[0].approvalId, {
	approved: true,
	feedback: "Looks right.", // optional; especially useful on a denial
	remember: false, // true => stop prompting for this tool on future runs
});
// → { status: "recorded" }
```

- **Render `toolName` / `toolArgs`** so the user sees exactly what's about to run before they approve; use `description` for a human-readable summary when present.
- **`remember: true`** persists the decision agent-wide so the tool stops prompting on later runs — a no-op for tools that aren't permission-governed. Defaults to `false`.
- **Don't tight-loop `listPendingApprovals`.** Poll on a sensible cadence (thread open, or a cron) — the run is already blocked, so there's nothing to race.

## Error handling

The package throws typed errors — branch on them rather than parsing messages:

- **`AgentRunNotActiveError`** (from `sendMessage` on a finished run): the run has completed and the pod is gone. Start a fresh run with `run()` rather than retrying the message.
- **`AgentNotFoundError`**: unknown run/agent id — also thrown by `respondToApproval` when the approval is no longer pending (already answered, or it expired and auto-denied). Re-fetch with `listPendingApprovals` and re-render.
- **`AgentsAuthError`**: the runner lacks `agent:use`, or the generated client is missing / stale. Re-run `sandbox_add-agent-client`.
- **`AgentsValidationError`**: bad input (missing `prompt` / `message`).

```typescript
import { AgentRunNotActiveError } from "@major-tech/agents-client";

try {
	await supportBotClient.sendMessage(runId, text);
} catch (err) {
	if (err instanceof AgentRunNotActiveError) {
		const { chatThreadId } = await supportBotClient.run({ prompt: text });
		return chatThreadId;
	}
	throw err;
}
```

## Tips

- **`runId` === `chatThreadId`** — the value `run()` returns is what every run-op takes. Don't invent a separate id.
- **Runs are async.** Don't expect output from `run()`. Render a pending state, then poll `getAgentContent` (or open the agent side-by-side) for results.
- **`getAgentContent` / `getRunningInstancesOfAgent` are always safe** to call regardless of run state. `sendMessage` / `stopAgent` only make sense on your own runs — a run-op against another app's run is rejected.
- **`AgentMessage.content` shape varies by `type`** (`message`, `thinking`, `tool_use`, `tool_result`, `result`, system types). Render defensively.
- **Don't loop-spawn runs.** Each run costs credits and spins up a real agent session, so kick off only the few runs the page actually needs and let them finish. For a command center, that's the handful of agents you're summarizing — never start runs in a render loop or recursively.
