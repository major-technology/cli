---
name: workflow-builder
description: Create and manage Major workflows — JSONC graphs of agent calls, app calls, routers, loops, joins, waits, and human approvals fired by schedules, connector events, or authenticated webhooks — the definition format, node/edge/trigger reference, state expressions, and the sandbox-based edit/validate/save/publish lifecycle.
---

# Building a Major workflow

A _workflow_ is a graph of steps executed by Major's workflow engine: agents run with prompts, deployed apps get called over HTTP, routers branch on state, loops fan over collections, humans approve over Slack, and schedules, connector events, or authenticated webhooks start the graph. You author the definition as a JSONC file (JSON with comments) on the workflow's sandbox and edit it through the sandbox tools (the "Working with sandboxes" section of your system prompt covers addressing, provisioning, and sharing).

Orchestrator tools are `mcp__orchestrator-platform__*` (`list_workflows`, `mount`, `create_workflow`, `publish`, `run_workflow`, `list_workflow_runs`, `get_workflow_run`, `delete_workflow`, `list_connector_event_types`). File editing and sync go through the sandbox tools `mcp__sandbox__*` (`read_file`, `edit_file`, `write_file`, `pull`, `push`, `validate`), each called with `workflow: "<workflowId>"` as the target. `publish` takes the same `workflow` argument.

## The working file, saving, and publishing

Each workflow's working copy lives on its own sandbox at `<workflowId>.jsonc` (workspace-relative). Two separate steps take it off the sandbox:

- **Save** (`push`) writes the file as a new immutable version. Nothing about the workflow's behaviour changes — saving is free. **The sandbox is torn down once it goes idle and comes back seeded from the last saved version, so anything unsaved is lost.**
- **Publish** (`publish`) points the workflow at its latest saved version. This is the only thing that changes live trigger behavior: schedules begin firing, connector endpoints are reconciled, and webhook URLs are materialized.

So triggers sitting in a saved-but-unpublished draft are inert, and `run_workflow` runs the last **saved** version — you never have to publish to test.

Always use a `workflowId` returned by `list_workflows` or `create_workflow` — never invent one. If this chat is pinned to a workflow, the "Working with this workflow" section of your system prompt carries the bound-chat rules (its id, recovery, discard) — those win.

## Save discipline

- **Never `pull` routinely** — it overwrites the sandbox file with the last saved version and destroys any unsaved edits, the user's included. Pull only to recover a corrupted file or on the user's explicit ask to discard.
- **Always `push` before you finish a turn in which you edited the file.** Unsaved work dies with the sandbox. Saving needs no permission and changes nothing about how the workflow runs.
- **Publish only when the user asks for it.** That is the moment the workflow starts acting on its own.

## Lifecycle

- **Edit existing**: `list_workflows` to find it, then `mount({workflow: "<workflowId>"})` — it mounts (or joins) the workflow's sandbox and returns the file name. Edit the file with the sandbox tools, saving as you finish each round.
- **New**: `create_workflow({})` — it creates a skeleton workflow (server-minted `workflowId`), mounts its sandbox, and returns the file name. Build the definition in that file; there is no push-a-loose-draft path.
- **Check a draft**: `validate` — validates the sandbox file against the server's rules without saving. Saving also validates; on failure nothing is saved and the error list comes back — fix the file and save again.
- **Save**: `push` — every save writes a new immutable version; comments are preserved verbatim.
- **Test**: `run_workflow` (save first — it runs the last saved version), then `get_workflow_run` — it returns the per-node trace (status, resolved input, output, errors). App calls use deployed apps by default. Pass `appTarget: "sandbox"` to test `app_call` nodes against the acting user's live app sandboxes (spun up on demand) before deploying; a sandbox held by another user fails that node with the holder's name.
- **Publish**: `publish({workflow})` — makes the latest saved version live. Name the target the same way the sandbox tools do: pass the `workflowId` you passed to `push`. Only on the user's explicit go-ahead. Add `versionId` to roll back to an earlier version.

## The definition format

The definition shape AND the enforced graph rules are the workflow-definition JSON Schema the Major API serves at `GET https://api.prod.major.build/public/workflow.schema.json` — the single source of truth (the graph rules are its `x-validatorRules`). YOU MUST CURL THIS SCHEMA BEFORE BUILDING A WORKFLOW. `validate` (and every push) enforces all of it, with errors naming the offending path.

Runtime semantics the schema can't express:

- **router**: branches evaluate top to bottom; the first `when` `$expr` that is true wins, else `default` fires. A branch's `to` array activates ALL its targets in parallel.
- **for_each**: the body edge is traversed once per item of `config.input.iterable`; `mode: "parallel"` runs iterations concurrently (bounded by `max_concurrency`), default is sequential. Loop bodies are the only legal cycles.
- **join** (`mode: "all"`): releases once every inbound edge has arrived. Keep fan-out legs simple lines into the join — the validator rejects shapes it can't prove safe.
- **human_approval**: point its single out-edge at a `router` that branches on `<node_id>.choice`, otherwise the decision can't affect the flow.
- Prefer underscores over hyphens in ids — a hyphenated node id can't be referenced from `$expr`.

## Trigger input

`input_schema` is optional. When supplied, it must be a standard JSON Schema whose root describes an object; express required properties with `required` and nullable values with a type union such as `["string", "null"]`. Before every Run Now, cron, connector-event, or webhook run, the platform recursively materializes schema defaults, merges the trigger-produced input over them (explicit values, including `null`, win; arrays replace arrays), and validates without coercion. Without `input_schema`, no defaults or validation are applied.

A webhook trigger is `{ "id": "...", "type": "webhook" }` with optional `label` and `input`; it has no `config`. Adding one in the workflow editor immediately creates its public URL and reveals its credential once. A webhook authored through JSON or MCP receives its URL at publish; the user must then open the workflow and click **View credential** to generate and reveal its credential. Credentials never appear in workflow JSON or MCP results: do not ask for, retrieve, echo, or place them in chat. Never claim a webhook is ready until the user confirms they received its one-time setup details.

Webhook `$expr` reads only `event.json`. Omit or empty `input` to forward a JSON-object body as `trigger.input`; a non-empty `input` is the same literal/`$expr` projection as connector events.

A connector event's `options.events` must be a non-empty, duplicate-free list. One trigger-level `input` projection is shared by every listed event; there are no per-event input overrides. Omitted or empty `input` forwards the complete normalized `event.json` object as flat `trigger.input`; a non-empty `input` explicitly restricts or reshapes those fields. Projection leaves are JSON literals or `{ "$expr": "..." }`. Projection CEL sees only `event.id`, `event.type`, optional `event.occurred_at`, and `event.json`; it does not see nodes or `trigger`. Call `list_connector_event_types` with the intended `connectorType` to discover concise supported event names and the required resource subtype. After choosing an event, call it again with `eventType` to inspect that event's focused `payloadSchema` for normalized `event.json`; do not request payload details for events you are not using. An empty event list means Major does not support connector events for that type. Then use the id of a configured matching resource as `resource_id` — never guess these values.

### Connector-event filters

`config.options.filter` is an optional boolean CEL expression evaluated before input projection, using the same `event.id`, `event.type`, optional `event.occurred_at`, and `event.json` namespace. Inspect each selected event's `payloadSchema` with `list_connector_event_types`, put deterministic eligibility such as channel, subtype, or amount in the filter, and guard optional fields before reading them. Never start an agent for every event and tell it to do nothing for unwanted events.

```jsonc
// Slack: top-level messages in one channel
"filter": "event.json.channel == 'C0123456789' && !('thread_ts' in event.json) && !('subtype' in event.json)"

// Stripe: successful USD payments of at least $100
"filter": "event.json.amount >= 10000 && event.json.currency == 'usd'"
```

Slack bot/app messages are excluded by default.

## Referencing state

Each node's output is stored under its node id; any later node's `config.input` / `prompt` / `message` / `output` can reference it:

- `{"$state": "node_id.path.to.value"}` — direct lookup (array indices allowed: `items.0.name`). A missing `$state` path fails the run. Use `trigger.input.<field>` for canonical workflow input.
- `{"$expr": "<CEL expression>"}` — computed values; node ids are variables (un-run nodes are `null`), plus `coalesce(a, b, ...)` (2–5 args) and `count(x)`. `$expr` is lenient: an expression that touches a missing path evaluates to `null` (a router condition that is `null` simply doesn't match) — use `has(node_id.field)` to branch on optional fields.
- `"{{node_id.path}}"` inside strings — interpolation; `"{{json node_id.path}}"` serializes objects/arrays.

Node outputs by type:

- `app_call` sends its resolved `config.input` as query params for GET/DELETE and as a JSON body for POST/PUT/PATCH; its output is the response body (non-object bodies wrap as `{"body": ...}`).
- `agent_call` with an `output_schema` outputs exactly that shape; without one it outputs `{"result": "<final message text>"}`.
- `human_approval` outputs `{"choice": "<options[].value>", "responded_by": "<user id>", "timed_out": true?}` (`responded_by` absent on timeout).
- Inside a `for_each` body, `<for_each_id>.item` and `<for_each_id>.index` are the current iteration; after the exit edge, `<for_each_id>.results` holds the per-iteration results in item order (the output of each iteration's final node) unless `collect_results` is false.
- `router`, `join`, and `wait` produce no state worth referencing.

## Workflow

1. Ask what the workflow should do, which agents/apps it touches (`list_agents`, `list_use_apps` / `list_edit_apps` to discover ids), and the cadence.
2. `create_workflow` (or `mount` for an existing one) — the builder panel opens so the user can see the graph.
3. Draft the JSONC. Iterate with the sandbox file tools, `validate` as you go, and `push` at the end of every turn you edited in.
4. Test with `run_workflow` (it runs what you last saved), then inspect with `get_workflow_run`.
5. Write the requested cron, connector event, or webhook into the file once its configuration is known. An unpublished trigger is inert, so it costs nothing to save. Never invent an automated trigger the user didn't ask for.
6. Once the user confirms it's ready, publish it. For a newly added webhook, ask the user to open the workflow editor after publish and click **View credential** so the browser can show its one-time setup details. For other changes, `publish({workflow})` makes the saved version live.

To pause a live workflow, remove or comment out its trigger, save, and publish. `delete_workflow` only on explicit user request.

## Approvals (Slack-only)

Two distinct approval surfaces, both needing a connected Slack resource. Check first: `mcp__resources__execute_resource_tool` with `toolName: "mcp__resources__list_resources"` — if no Slack resource exists, Slack isn't connected; say so (it's provisioned by installing the Major Slack integration under Settings → Integrations) instead of asking them to pick a channel. To find a channel id, execute the Slack resource's `mcp__resources__slack_list_channels` the same way.

- A **`human_approval` node** is a first-class step: it posts `config.message` to `config.channel` (`{"type": "slack", "channel_id": ...}`) with `options` as buttons and the run waits for the click (or `on_timeout`). Use it whenever the user wants a person to sign off mid-flow.
- An `agent_call`'s `approval_channel` (`{"type": "slack", "channel_id": ..., "channel_name": ...}`) routes that agent session's tool-approval requests to Slack; omitted/null keeps approvals in the app. Don't pitch Slack routing unprompted.

## Style

If you don't have enough information to pick agents, apps, paths, or a cron, ask — never guess ids. Keep the JSONC readable: comments where intent isn't obvious, labels on nodes and branch edges (they render in the visualizer). Be terse — the builder panel mirrors the sandbox file live, so don't echo the definition back into chat.
