---
name: app-builder
description: Create and edit Major apps — full-stack Next.js apps with frontend, backend API routes, and a live preview — how to create or mount an app sandbox and edit through the workspace tools.
---

# Building a Major app

An _app_ on Major is a full-stack Next.js app: frontend and backend API routes, deployed and hosted by Major. Apps can use any connector. Agents and workflows call a deployed app's API directly (Major handles the auth and plumbing).

Treat apps as **compute**. Whenever you need to run code, use an app. Apps also visualize results for the user. Offload deterministic behavior into the app; agents and workflows call it.

The main way to work on an app is to mount its sandbox onto this chat and edit it yourself. Finding, creating, opening, and closing apps is on `mcp__plugin_major-build_major__*` (`list` and `create` with `target_type: "app"`, `list_sandboxes` for what is mounted, `start_sandbox`, `stop_sandbox`). File and shell work go through the `mcp__plugin_major-build_major__sandbox_*` tools — the "Working with sandboxes" section of your system prompt covers the tools, argument conventions, provisioning, shared-sandbox etiquette, and local-file uploads; an app's target argument is `slug`. Run `major` CLI commands through `mcp__plugin_major-build_major__sandbox_bash` inside the mounted app workspace; they infer the app from the working directory, so do not pass an app ID or run them in the agent's local filesystem. Deployment and app-to-agent wiring still use major-app MCP tools.

You are in general chat — **nothing is bound to this thread**, and a sandbox is not always mounted. Always use an `applicationId` or `slug` returned by `list` or `create` (`target_type: "app"`) — never invent one. If the app you need is not in `list_sandboxes`, start it (`start_sandbox({slug: "<slug>"})`, or `app: "<applicationId>"`) or create it (`create({target_type: "app", name, description})`) before using `mcp__plugin_major-build_major__sandbox_*` tools.

If this chat is already pinned to an app (the "Working with this app" section of your system prompt), its sandbox is auto-mounted — skip create/mount and edit that app through the sandbox tools with its slug.

## Lifecycle

- **Edit existing**: `list({target_type: "app"})` to find it, then `start_sandbox({slug: "<slug>"})`. This wakes the app's sandbox, attaches it to this chat, and starts a live preview. A `locked` result means another user holds the app — tell the user who; do not retry in a loop.
- **New**: `create({target_type: "app", name, description})` with a short name and a one-sentence description. It returns the new `applicationId` and automatically mounts the sandbox — you do **not** need to call `start_sandbox`. For a brand-new app's first iteration, load the `new-project` skill and follow it before writing code.
- **Save**: commit and push on `main` using the sandbox shell tool. Stage only the files you changed (`git add <paths>`) — never `git add -A` or `git add .`: other chat sessions may be editing the same workspace. Never run `git stash` (or `git stash push` / `git stash pop` / `git stash apply`). Never create feature branches.
- **Deploy**: a separate, explicit step — do **not** call `deploy_app` unless the user asked to deploy/publish/ship in this conversation. Finishing an edit means committing and pushing on `main`, then telling the user the change is ready to deploy. When they do ask, batch all finished changes into a single deploy. A deploy builds for ~2 minutes — tell the user it is building and end your turn; never poll `major app info` in a loop.

If a request is ambiguous (you can't tell which existing app it maps to, or you lack the detail to mount it), ask one or two clarifying questions first.

## Build rules

The preview dev server is ALREADY running in the sandbox and hot-reloads on save — you do NOT need to build to see changes, and the preview is what the user sees.

- NEVER run `next build`, `pnpm build`, `npm run build`, or `yarn build`. Only run a build if you are specifically debugging a build issue.
- NEVER delete or remove the `.next` directory — it crashes the preview server and the entire session.
- To check for errors, run lint (through `mcp__plugin_major-build_major__sandbox_bash`) instead of building. After you finish editing, always run a lint check and fix what it reports — lint failures will fail a deploy. Lint ONCE per finished change, not after every file edit.
- When using parallel subagents, each subagent should ONLY write code and run lint. Do NOT have subagents run build commands.

The preview/sandbox runtime has these environment variables available — use `mcp__plugin_major-build_major__sandbox_bash` with curl to hit the APIs you write:

- `MAJOR_API_BASE_URL` — the base url of the Major API
- `MAJOR_JWT_TOKEN` — the JWT token for the Major API
- `APPLICATION_ID` — the id of the application

## Working efficiently

Every tool result you pull into this chat is re-read on each later step, so keep results small:

- Don't re-read files you just read or wrote — the content is already in your context. For large files, page with `mcp__plugin_major-build_major__sandbox_read_file`'s offset/limit instead of re-reading the whole file.
- Push bulk lookups to subagents and have them return conclusions only: database verification queries (postgresql_psql), broad code exploration, and log digging. Don't run row-dump queries in the main chat.
- Do not verify the UI unless explicitly asked. Playwright verification is costly and should be used sparingly.
- When dispatching a subagent (or running a workflow of subagents), explicitly select its model instead of leaving it unset. Prefer a smaller/cheaper model (e.g. haiku) for routine work — bulk lookups, log digging, simple code exploration, mechanical edits — and reserve a larger model for tasks that genuinely need deeper reasoning (architecture decisions, tricky debugging, ambiguous requirements).

## Browser QA

Never open the browser, take screenshots/snapshots, check console via browser tools, or dispatch `browser-qa` unless the user explicitly asked for visual verification in this conversation (e.g. "check the UI", "take a screenshot", "verify it looks right", "does the page render?"). Editing React/UI code, changing component props, finishing a feature, or "making sure it works" is NOT a reason — lint is enough; the live preview is already what the user sees.

When (and only when) the user asked, dispatch the `browser-qa` subagent with the app slug, the route to check, and the specific things to verify — never drive the browser tools from the main chat; page snapshots are large and permanently bloat this conversation.

## Plan mode

When launching subagent tasks for planning, instruct them to read the project's agent guide (`AGENTS.md`, or `CLAUDE.md` if that's all that exists) — it carries vital information about the app — and to NOT call the exit-plan-mode tool — only the main agent exits plan mode.

To exit plan mode, first write your complete plan to a LOCAL file using your built-in Write tool (an absolute path, e.g. /tmp/plan.md). Then call `mcp__plan-mode__exit-plan-mode` with planFilePath set to that absolute path. Use your built-in Write tool for the plan file, NOT the sandbox file tools — the plan must live on your local filesystem so exit-plan-mode can read it; it is not part of the app code.

## Frontend design

Run `major app theme get` in the app workspace before frontend work. It returns the app's design system: colors, font, border radius, and logo (full and/or small version, when provided). Use only the parameters the theme provides unless the user explicitly asks for a custom design. If it reports no theme, the app has no configured theme yet. Use `major app theme list` to discover themes and `major app theme apply <themeId>` to select one; applying also writes the theme files into the checkout.

## Debugging & agent-triggering playbooks

Use whichever applies before you start:

- Load the `debug-issue` skill whenever you're investigating a failure, regression, or broken/blank/errored behavior in the app (covers the preview, app errors, logs, and browser inspection).
- Read [references/using-agents.md](references/using-agents.md) (in this skill's directory) when wiring the app's runtime code to trigger Major agents (run / sendMessage / stop / approvals; `sandbox_add-agent-client` generates the typed client, same pattern as resource clients).
- To call another app in the org, run `major app-client add --id <appId>` (ids from `major app list`), then `await <name>Fetch('/path')` from server code.

## Recurring work

Apps no longer carry their own crons. `cron.json` is not read. For recurring work against an app's API, load the `workflow-builder` skill and build a workflow with a cron trigger and an `app_call` node.

## Calling go-api from app code

Do not construct `PostgresResourceClient`, `SlackResourceClient`, `createProxyFetch`, or any other resource client by hand. Import the generated client in `clients/`. It already copies the incoming `x-major-user-jwt`.

A resource call to go-api without `x-major-user-jwt` will be rejected. A hand-rolled client that only sets `MAJOR_JWT_TOKEN` fails, including from a webhook or a workflow `app_call`.

A webhook route and a workflow `app_call` are real requests. Ingress has already set `x-major-user-jwt`. `headers()` works there. Do not drop `getHeaders` so those routes can run.

`MAJOR_JWT_TOKEN` alone is only for the runner and for code that is not inside a request. `current-build` and the error reporter keep using it.

## LLM calls from app code

App code can call LLMs through Major's AI proxy — no API key needed, spend is metered per app. Check `major app ai-proxy status` first, then run `major app ai-proxy enable` with the user's go-ahead. Enabling sets a default $10/month limit.

## Inspecting

Run these commands in the mounted app workspace through `mcp__plugin_major-build_major__sandbox_bash`:

- `major app info --json` — deployment status, the deployed URL, and visibility
- `major app logs --preview` — preview/dev-server output
- `major app logs` — deployed app logs
- `major app errors list` / `major app errors get <errorId>` — inspect runtime errors
- `major app errors resolve <errorId>` — after committing a fix for a confirmed runtime error
- `major app errors enable` — after adding the Major error-reporter scaffolding to the repo

Use each command's `--help` for filters and pagination.

## User setup

For app secrets, use the available MCP setup tool: `set-app-env-variables` in app chats, or `set_env_variables` on the build server. The user supplies values through the frontend; never ask them to paste secrets into chat. If the tool returns a setup URL, share it in one short sentence and wait for the user's confirmation. Use `major vars set KEY=VALUE` only for known values the user explicitly wants you to configure.

New connector setup still uses `mcp__plugin_major-build_major__request_resource_setup`; load `using-connectors` for that flow. An existing connector can be added to app code separately.
