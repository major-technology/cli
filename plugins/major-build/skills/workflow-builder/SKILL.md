---
name: workflow-builder
description: Create and manage Major workflows — JSONC graphs of functions, Jev judgment steps, agent calls, app calls, routers, loops, joins, waits, and human approvals fired by schedules, connector events, or authenticated webhooks — the definition format, node/edge/trigger reference, state expressions, and the sandbox-based edit/validate/save/publish lifecycle.
---

# Building a Major workflow

A _workflow_ is a graph of steps executed by Major's workflow engine: the workflow's own TypeScript functions run compute, agents run with prompts, deployed apps get called over HTTP, routers branch on state, loops fan over collections, humans approve over Slack, and schedules, connector events, or authenticated webhooks start the graph. You author the definition as a JSONC file (JSON with comments) on the workflow's sandbox and edit it through the sandbox tools (the "Working with sandboxes" section of your system prompt covers addressing, provisioning, and sharing).

Finding, creating, opening, and running workflows is on `mcp__plugin_major_major__*` (`list` and `create` with `target_type: "workflow"`, `start_sandbox`, `run_workflow`). File editing goes through the sandbox tools `mcp__plugin_major_major__sandbox_*` (`sandbox_read_file`, `sandbox_edit_file`, `sandbox_write_file`, `sandbox_bash`), each called with `target_type: "workflow"` and `target_id: "<workflowId>"` as the target. Sync and publishing are the `major` CLI, run through `sandbox_bash` in `/workspace/workflow`: `major pull`, `major push -m "<notes>"`, `major validate`, `major publish --yes`.

Inspect runs and connector events with the CLI, through `sandbox_bash` in `/workspace/workflow`: `major workflow run list`, `major workflow run get <runId>` (per-node trace), and `major workflow events <connectorType> [--event <type>]` (connector event types and payload schemas).

## The working file, saving, and publishing

Each workflow's working copy lives on its own sandbox at `workflow.jsonc` (workspace-relative). Two separate steps take it off the sandbox:

- **Save** (`major push`) saves every function under `functions/`, then writes the file and the rest of the workspace as a new immutable version. Nothing about the workflow's behaviour changes — saving is free. **The sandbox is torn down once it goes idle and comes back seeded from the last saved version, so anything unsaved is lost.**
- **Publish** (`major publish --yes`) publishes every function under `functions/`, then points the workflow at its latest saved version. This is the only thing that changes live trigger behavior: schedules begin firing, connector endpoints are reconciled, and webhook URLs are materialized.

So triggers sitting in a saved-but-unpublished draft are inert, and `run_workflow` runs the last **saved** version, with each function's latest saved version — you never have to publish to test.

Always use a `workflowId` returned by `list` or `create` (`target_type: "workflow"`) — never invent one. If this chat is pinned to a workflow, the "Working with this workflow" section of your system prompt carries the bound-chat rules (its id, recovery, discard) — those win.

## Save discipline

- **Never `major pull` routinely** — it overwrites the sandbox file with the last saved version and destroys any unsaved edits, the user's included. Pull only to recover a corrupted file or on the user's explicit ask to discard.
- **Always `major push` before you finish a turn in which you edited the file.** Unsaved work dies with the sandbox. Saving needs no permission and changes nothing about how the workflow runs.
- **Publish only when the user asks for it.** That is the moment the workflow starts acting on its own.

## Lifecycle

- **Edit existing**: `list({target_type: "workflow"})` to find it, then `start_sandbox({target_type: "workflow", target_id: "<workflowId>"})` — it mounts (or joins) the workflow's sandbox and returns the file name. Edit the file with the sandbox tools, saving as you finish each round.
- **New**: `create({target_type: "workflow"})` — it creates a skeleton workflow (server-minted `workflowId`), mounts its sandbox, and returns the file name. Build the definition in that file; there is no push-a-loose-draft path.
- **Check a draft**: `major validate` — validates the sandbox file against the server's rules without saving. Saving also validates; on failure nothing is saved and the error list comes back — fix the file and save again.
- **Save**: `major push -m "<notes>"` — every save writes a new immutable version; comments are preserved verbatim.
- **Test**: `run_workflow` (save first — it runs the last saved version), then `major workflow run get <runId>` — it returns the per-node trace (status, resolved input, output, errors). App calls use deployed apps by default. Pass `appTarget: "sandbox"` to test `app_call` nodes against the acting user's live app sandboxes (spun up on demand) before deploying; a sandbox held by another user fails that node with the holder's name.
- **Publish**: `major publish --yes` — makes the latest saved version live. Only on the user's explicit go-ahead.

## The definition format

The definition shape AND the enforced graph rules are the workflow-definition JSON Schema the Major API serves at `GET ${MAJOR_URL:-https://api.prod.major.build}/public/workflow.schema.json` (`MAJOR_URL` is your environment's Major API when it's set) — the single source of truth (the graph rules are its `x-validatorRules`). YOU MUST CURL THIS SCHEMA BEFORE BUILDING A WORKFLOW. `major validate` (and every push) enforces all of it, with errors naming the offending path.

Runtime semantics the schema can't express:

- **router**: branches evaluate top to bottom; the first `when` `$expr` that is true wins, else `default` fires. A branch's `to` array activates ALL its targets in parallel.
- **for_each**: the body edge is traversed once per item of `config.input.iterable`; `mode: "parallel"` runs iterations concurrently (bounded by `max_concurrency`), default is sequential. Loop bodies are the only legal cycles.
- **join** (`mode: "all"`): releases once every inbound edge has arrived. Keep fan-out legs simple lines into the join — the validator rejects shapes it can't prove safe.
- **human_approval**: point its single out-edge at a `router` that branches on `<node_id>.choice`, otherwise the decision can't affect the flow.
- Prefer underscores over hyphens in ids — a hyphenated node id can't be referenced from `$expr`.

## Choosing a step: the cheapest node that can do it

An agent run costs about 100 times a judgment step, and a function costs almost nothing. Give each step to the first of these that can do it:

- **`router`/`$expr`**: simple branching or a one-line computed value.
- **`function`**: TypeScript the workflow owns and versions with it, and the default for anything code can do: fetching (new issues, unread mail, the meetings that just ended, rows matching a query), filtering, aggregating, formatting, and writing to a connected resource or external API (apply a label, post a fixed message, update a record). Don't build or extend an app just to give a workflow somewhere to run code.
- **`app_call`**: only when the compute belongs to an app — the user names an existing app endpoint to call, the logic already lives in a deployed app, or the step changes state an app owns (its data, its jobs) and the app should stay the one place that does it. If no such app exists, write a function instead.
- **`workflow_call`**: a step that is another workflow's whole job — call it instead of copying its nodes. It runs that workflow's published version (publish it first), as that workflow's deployer. Find ids with `list` (`target_type: "workflow"`, `include_read_only: true`).
- **`jev_noul` / `jev_choice` / `jev_score`**: a judgment about text the run already holds — yes/no questions ("does this email need a reply today?"), one label from a list, or a score. Each takes `text`, a template over run state, and sees nothing else: it can't fetch context or use anyone's memory, so put everything it needs in `text` and write explicit criteria. Route on its output with a `router`.
- **`agent_call`**: only what needs reasoning across several steps, tools chosen as it goes, or drafting.

**Gate every agent.** A run that turns out to have no work must end without starting one, so an agent is never the entry node of a triggered workflow unless every trigger genuinely needs its judgment:

- When the run begins by checking for work, a function does the check and a `router` reaches the agent only when it found some, otherwise an `end` node. A function keeps nothing between runs: on a cron trigger, derive "new since the last run" from a time window as long as the schedule's interval.
- When whether to act is a judgment about the trigger's content, a Jev node decides and the `router` reads it. When that judgment is all the agent would have done (classify then label, decide then notify), Jev decides and a function or `pepper_call` carries it out, with no agent at all.
- Put what the function fetched and Jev decided into the agent's prompt, so the agent doesn't fetch or decide it again.
- Don't add a gate that always passes: it only adds cost.

```jsonc
// Hourly: an agent files action items only when a meeting actually ended.
{ "id": "ended_meetings", "type": "function", "label": "Meetings that just ended", "config": { "function": "ended-meetings", "input": {} } },
{ "id": "any_ended", "type": "router", "label": "Any ended?", "config": { "branches": [{ "label": "Yes", "when": { "$expr": "count(ended_meetings.meetings) > 0" }, "to": ["file_actions"] }], "default": ["done"] } },
{ "id": "file_actions", "type": "agent_call", "label": "File action items", "config": { "agent_id": "<agentId>", "prompt": "File the untracked action items from these meetings: {{json ended_meetings.meetings}}" } }
```

### Writing a function

A function is its own versioned thing inside the workflow: a self-contained folder `functions/<slug>/` next to the definition file. Create it with the CLI, never by hand — a bare folder has no function id and fails the workflow's `major push`:

```bash
major function create functions/<slug>   # <slug> matches ^[a-z][a-z0-9-]{0,62}$
```

The folder holds everything the function needs:

- `index.ts` — `export default async function (input) { ... }`; JSON in, JSON out.
- `function.json` — required. `{ "label", "description", "input_schema", "output_schema" }`, each field optional. A schema that's present is enforced on every call (bad input or output fails the node), so supply both.
- `package.json` — the function's own npm dependencies; run `npm install` in the function folder before testing. `node_modules` isn't saved: a push bundles the function with its dependencies inlined.
- `clients/` and `resources.json` — generated resource clients (below). Nothing is shared between functions or with the workflow root.

```jsonc
{ "id": "score_leads", "type": "function", "label": "Score leads", "config": { "function": "score-leads", "input": { "leads": { "$state": "fetch_leads.rows" } }, "max_retries": 2 } }
```

- `config.input` resolves like any node's input and is the function's argument. A thrown error or timeout is retried with backoff up to `max_retries` (0–10). Calls time out after 30s, so keep functions short; split long work across a `for_each`.
- To use a resource, generate a typed client into the function with `sandbox_add-resource-client({resourceId, resourceName, resourceType, resourceDescription, functionSlug: "<slug>", target_type: "workflow", target_id: "<workflowId>"})` and use the import line it returns verbatim — never hand-write client or auth code. The function calls resources as the run's user (the publisher on a triggered run). It also has open outbound network access.
- `major validate`, `major push`, and `major publish --yes` in the workflow root run on every function first, in parallel, then on the workflow; a function that doesn't build fails the command and names the file and line. Run the same commands inside `functions/<slug>/` to act on that one function alone.
- **Published runs use each function's published version, not the one the workflow was saved with.** Publishing a function changes the live workflow at once, without republishing the workflow, and rolling the workflow back to an older version does not roll its functions back. Publish a function on its own only on the user's explicit go-ahead.
- Renaming a folder does not rename the function: create a new one under the new name and move the code.

## Trigger input

`input_schema` is optional. When supplied, it must be a standard JSON Schema whose root describes an object; express required properties with `required` and nullable values with a type union such as `["string", "null"]`. Before every Run Now, cron, connector-event, or webhook run, the platform recursively materializes schema defaults, merges the trigger-produced input over them (explicit values, including `null`, win; arrays replace arrays), and validates without coercion. Without `input_schema`, no defaults or validation are applied.

A webhook trigger is `{ "id": "...", "type": "webhook" }` with optional `label` and `input`; it has no `config`. Adding one in the workflow editor immediately creates its public URL and reveals its credential once. A webhook authored through JSON or MCP receives its URL at publish; the user must then open the workflow and click **View credential** to generate and reveal its credential. Credentials never appear in workflow JSON or MCP results: do not ask for, retrieve, echo, or place them in chat. Never claim a webhook is ready until the user confirms they received its one-time setup details.

Webhook `$expr` reads only `event.json`. Omit or empty `input` to forward a JSON-object body as `trigger.input`; a non-empty `input` is the same literal/`$expr` projection as connector events.

A connector event's `options.events` must be a non-empty, duplicate-free list. One trigger-level `input` projection is shared by every listed event; there are no per-event input overrides. Omitted or empty `input` forwards the complete normalized `event.json` object as flat `trigger.input`; a non-empty `input` explicitly restricts or reshapes those fields. Projection leaves are JSON literals or `{ "$expr": "..." }`. Projection CEL sees only `event.id`, `event.type`, optional `event.occurred_at`, and `event.json`; it does not see nodes or `trigger`. Run `major workflow events <connectorType>` to discover concise supported event names and the required resource subtype. After choosing an event, run it again with `--event <type>` to inspect that event's focused `payloadSchema` for normalized `event.json`; do not request payload details for events you are not using. An empty event list means Major does not support connector events for that type. Then use the id of a configured matching resource as `resource_id` — never guess these values.

### Connector-event filters

`config.options.filter` is an optional boolean CEL expression evaluated before input projection, using the same `event.id`, `event.type`, optional `event.occurred_at`, and `event.json` namespace. Inspect each selected event's `payloadSchema` with `major workflow events <connectorType> --event <type>`, put deterministic eligibility such as channel, subtype, or amount in the filter, and guard optional fields before reading them. Never start an agent for every event and tell it to do nothing for unwanted events; eligibility that needs a judgment about the content goes to a Jev node (see "Gate every agent").

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

- `function` outputs its return value when that is an object; anything else, including `null` when it returns nothing, wraps as `{"body": ...}`.
- `app_call` sends its resolved `config.input` as query params for GET/DELETE and as a JSON body for POST/PUT/PATCH; its output is the response body (non-object bodies wrap as `{"body": ...}`).
- `agent_call` with an `output_schema` outputs exactly that shape; without one it outputs `{"result": "<final message text>"}`.
- `workflow_call` (`config: {"workflow_id", "input"}`, `input` resolved like `app_call`'s and checked against the called workflow's `input_schema`) outputs the called run's end output (non-object outputs wrap as `{"result": ...}`); a failed or canceled called run fails the node.
- `pepper_call` outputs `{"messageId": "..."}`; nothing Pepper says comes back into the run.
- `jev_noul` outputs the yes-probability, 0 to 1, per question id (`<node_id>.<question_id>`; branch on `>= 0.5`). `jev_choice` outputs `{"choice", "confidence", "probabilities"}` and `jev_score` `{"score", "confidence", "probabilities"}`.
- `human_approval` outputs `{"choice": "<options[].value>", "responded_by": "<user id>", "timed_out": true?}` (`responded_by` absent on timeout).
- Inside a `for_each` body, `<for_each_id>.item` and `<for_each_id>.index` are the current iteration; after the exit edge, `<for_each_id>.results` holds the per-iteration results in item order (the output of each iteration's final node) unless `collect_results` is false.
- `router`, `join`, and `wait` produce no state worth referencing.

## Calling Pepper

Pepper is an organization member's always-on personal agent, which they chat with in Major and Slack. Use `pepper_call` only when your system prompt says the organization has Pepper. When the user says "my Pepper", they mean this agent, not one returned by `list`.

```jsonc
{ "id": "tell_pepper", "type": "pepper_call", "label": "Tell Pepper", "config": { "prompt": "New lead: {{json trigger.input}}" } }
```

- A triggered run's prompt goes to the publisher's Pepper; `run_workflow` and single-node runs go to the Pepper of whoever ran them. `prompt` interpolates state like an `agent_call` prompt.
- Pepper only tells its user, in its own chat, or decides nothing needs them. It never acts on a workflow's prompt (no emails, posts or changes for anyone), because other people can edit the workflow. Don't build a workflow that relies on Pepper doing something.
- The node fails when the organization has Pepper turned off.

## Workflow

1. Ask what the workflow should do, which agents/apps it touches (`list` with `target_type: "agent"` / `"app"` and `include_read_only: true` to discover ids), and the cadence. Plan each step on the cheapest node that can do it, and gate every agent (see "Choosing a step"). When an `app_call` needs endpoints or request/response shapes, load the `using-apps` skill.
2. `create({target_type: "workflow"})` (or `start_sandbox` for an existing one) — the builder panel opens so the user can see the graph.
3. Draft the JSONC. Iterate with the sandbox file tools, `major validate` as you go, and `major push` at the end of every turn you edited in.
4. Test with `run_workflow` (it runs what you last saved), then inspect with `major workflow run get <runId>`.
5. Write the requested cron, connector event, or webhook into the file once its configuration is known. An unpublished trigger is inert, so it costs nothing to save. Never invent an automated trigger the user didn't ask for.
6. Once the user confirms it's ready, publish it. For a newly added webhook, ask the user to open the workflow editor after publish and click **View credential** so the browser can show its one-time setup details. For other changes, `major publish --yes` makes the saved version live.

To pause a live workflow, remove or comment out its trigger, save, and publish.

## Approvals

A **`human_approval` node** is a first-class step: the run waits until a person picks one of `options` (or `on_timeout` fires). Use it whenever the user wants a person to sign off mid-flow. Pick `config.channel` by where the approver works:

- **Slack** (`{"type": "slack", "channel_id": ...}`): posts `config.message` with `options` as buttons.
- **An app** (`{"type": "app", "app_id": ..., "path": ...}`): POSTs `{runId, executionId, workflowId, message, payload, options}` to that endpoint of the app, `config.payload` resolved like `app_call`'s input, with `app_call`'s `max_retries`/`retry_on`. The app shows the request and answers with `approvals.respond({ runId, executionId, choice })` from `@major-tech/workflows-client`. Use it when the approver works in an app the user names; no reminders.

An `agent_call`'s `approval_channel` (`{"type": "slack", "channel_id": ..., "channel_name": ...}`) routes that agent session's tool-approval requests to Slack; omitted/null keeps approvals in the app. Don't pitch Slack routing unprompted.

Both Slack surfaces need a connected Slack resource. Check first: `mcp__plugin_major_major__execute_resource_tool` with `toolName: "mcp__resources__list_resources"` — if no Slack resource exists, Slack isn't connected; say so (it's provisioned by installing the Major Slack integration under Settings → Integrations) instead of asking them to pick a channel. To find a channel id, execute the Slack resource's `mcp__resources__slack_list_channels` the same way.

## Style

If you don't have enough information to pick agents, apps, paths, or a cron, ask — never guess ids. Keep the JSONC readable: comments where intent isn't obvious, labels on nodes and branch edges (they render in the visualizer). Be terse — the builder panel mirrors the sandbox file live, so don't echo the definition back into chat.
