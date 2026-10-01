---
name: app-builder
description: Create and edit Major apps — full-stack Next.js apps with frontend, backend API routes, and a live preview — how to create or mount an app sandbox and edit through the workspace tools.
---

# Building a Major app

An _app_ is a full-stack Next.js app (frontend + backend API routes) deployed and hosted by Major. Apps can use any connector; agents and workflows call a deployed app's API directly (Major handles auth and plumbing). Treat apps as **compute**: whenever you need to run code, use an app. Apps also visualize results for the user. Offload deterministic behavior into the app; agents and workflows call it.

## Tools

Below, `sandbox_*` means `mcp__plugin_major_major__sandbox_*`.

- **App lifecycle** — `mcp__plugin_major_major__*`: `list` / `create` (with `target_type: "app"`), `list_sandboxes` (what is mounted), `start_sandbox`, `stop_sandbox`.
- **Files and shell** — `sandbox_*`; an app's target argument is `slug`. The "Working with sandboxes" section of your system prompt covers the tools, argument conventions, provisioning, shared-sandbox etiquette, and local-file uploads.
- **`major` CLI** — run through `sandbox_bash` inside the mounted app workspace; it infers the app from the working directory. Never pass an app ID or run it in the agent's local filesystem.
- Deployment and app-to-agent wiring still use major-app MCP tools.

## Context

You are in general chat: **nothing is bound to this thread**, and a sandbox is not always mounted. Only use an `applicationId` or `slug` returned by `list` or `create` (`target_type: "app"`) — never invent one. If the app is not in `list_sandboxes`, start it (`start_sandbox({slug: "<slug>"})` or `app: "<applicationId>"`) or create it before using `sandbox_*` tools.

Exception: if this chat is pinned to an app (the "Working with this app" section of your system prompt), its sandbox is auto-mounted — skip create/mount and edit it through `sandbox_*` with its slug.

If a request is ambiguous (you can't tell which app it maps to, or lack the detail to mount it), ask one or two clarifying questions first.

## Lifecycle

1. **Edit existing**: `list({target_type: "app"})`, then `start_sandbox({slug: "<slug>"})` — wakes the sandbox, attaches it to this chat, starts a live preview. A `locked` result means another user holds the app: tell the user who; do not retry in a loop.
2. **New**: `create({target_type: "app", name, description})` with a short name and one-sentence description. Returns the `applicationId` and auto-mounts the sandbox — do **not** call `start_sandbox`. For a brand-new app's first iteration, load the `new-project` skill and follow it before writing code.
3. **Save** — ALWAYS commit and push; never leave changes uncommitted or unpushed. Commit and push on `main` via the sandbox shell. Stage only files you changed (`git add <paths>`) — never `git add -A` / `git add .` (other chat sessions may edit the same workspace). Never `git stash` in any form (`push`/`pop`/`apply`). Never create feature branches.
4. **Deploy** — separate, explicit step. Do **not** run `major publish --yes` unless the user asked to deploy/publish/ship in this conversation. Otherwise, finishing an edit = commit + push on `main`, then tell the user it is ready to deploy. When asked, batch all finished changes into one deploy: `major publish --yes` in `/workspace/app` (an app's first deploy needs `--slug <slug>` to choose its URL). The build takes ~2 minutes — tell the user it is building and end your turn; never poll `major app info` in a loop.

## Build rules

The preview dev server is ALREADY running in the sandbox and hot-reloads on save; the preview is what the user sees. No build needed to see changes.

- NEVER run `next build`, `pnpm build`, `npm run build`, or `yarn build` — unless specifically debugging a build issue.
- NEVER delete the `.next` directory — it crashes the preview server and the entire session.
- Check errors with lint (via `sandbox_bash`), not a build. After you finish editing, always lint and fix what it reports — lint failures fail a deploy. Lint ONCE per finished change, not per file edit.
- Parallel subagents ONLY write code and run lint — never build.

Sandbox env vars (use with curl via `sandbox_bash` to hit the APIs you write): `MAJOR_API_BASE_URL` (Major API base URL), `MAJOR_JWT_TOKEN` (Major API JWT), `APPLICATION_ID` (the application's id).

## Working efficiently

Every tool result is re-read on each later step, so keep results small:

- Don't re-read files you just read or wrote. For large files, page with `sandbox_read_file` offset/limit.
- Push bulk lookups to subagents that return conclusions only: database verification queries (postgresql_psql), broad code exploration, log digging. No row-dump queries in the main chat.
- Always set a subagent's model explicitly. Smaller/cheaper (e.g. haiku) for routine work — bulk lookups, log digging, simple exploration, mechanical edits; larger for architecture decisions, tricky debugging, ambiguous requirements.

## Browser QA

Playwright/browser verification is costly. Never open the browser, take screenshots/snapshots, check console via browser tools, or dispatch `browser-qa` unless the user explicitly asked for visual verification in this conversation (e.g. "check the UI", "take a screenshot", "verify it looks right", "does the page render?"). Editing React/UI code, changing props, finishing a feature, or "making sure it works" do NOT count — lint is enough.

When asked, dispatch the `browser-qa` subagent with the app slug, route, and specific things to verify. Never drive browser tools from the main chat — snapshots permanently bloat the conversation.

## Plan mode

- Planning subagents: instruct them to read the project's agent guide (`AGENTS.md`, or `CLAUDE.md` if that's all that exists) — it carries vital app information — and to NOT call exit-plan-mode; only the main agent exits.
- To exit: write the complete plan to a LOCAL file with your built-in Write tool (absolute path, e.g. `/tmp/plan.md`) — NOT the sandbox file tools; it's not app code and exit-plan-mode must read it locally. Then call `mcp__plan-mode__exit-plan-mode` with `planFilePath` set to that path.

## Frontend design

Before frontend work, run `major app theme get` in the app workspace. It returns the design system: colors, font, border radius, logo (full and/or small, when provided). Use only those parameters unless the user explicitly asks for a custom design. If it reports no theme, none is configured yet: `major app theme list` to discover themes, `major app theme apply <themeId>` to select one (also writes theme files into the checkout).

## Playbooks

- **Debugging** — load the `debug-issue` skill before investigating any failure, regression, or broken/blank/errored behavior (covers preview, app errors, logs, browser inspection).
- **Triggering agents** — before wiring app runtime code to Major agents (run / sendMessage / stop / approvals), read [references/using-agents.md](references/using-agents.md). `sandbox_add-agent-client` generates the typed client (same pattern as resource clients).
- **Recurring work** — apps no longer carry crons; `cron.json` is not read. Load the `workflow-builder` skill and build a workflow with a cron trigger and an `app_call` node.

## Calling go-api from app code

- Never construct `PostgresResourceClient`, `SlackResourceClient`, `createProxyFetch`, or any resource client by hand. Import the generated client from `clients/` — it already copies the incoming `x-major-user-jwt`.
- go-api rejects resource calls without `x-major-user-jwt`. A hand-rolled client that only sets `MAJOR_JWT_TOKEN` fails, including from a webhook or workflow `app_call`.
- Webhook routes and workflow `app_call`s are real requests: ingress already set `x-major-user-jwt` and `headers()` works. Do not drop `getHeaders` to make those routes run.
- `MAJOR_JWT_TOKEN` alone is only for the runner and code outside a request (`current-build` and the error reporter keep using it).

## LLM calls from app code

Use Major's AI proxy — no API key needed; spend is metered per app. Check `major app ai-proxy status` first; with the user's go-ahead, run `major app ai-proxy enable` (sets a default $10/month limit).

## Inspecting

Run in the mounted app workspace via `sandbox_bash`; use each command's `--help` for filters and pagination.

- `major app info --json` — deployment status, deployed URL, visibility
- `major app logs --preview` — preview/dev-server output; `major app logs` — deployed app logs
- `major app errors list` / `get <errorId>` — inspect runtime errors; `resolve <errorId>` — after committing a fix for a confirmed runtime error
- `major app errors enable` — after adding the Major error-reporter scaffolding to the repo
- `major resource invocations --slow [--environment coding-session] [--days 30]` — resource operations over 500 ms p95, ranked by total time, with call site; omit `--slow` to include faster ones

## User setup

- **App secrets**: `mcp__plugin_major_major__set_app_env_variables`. The user supplies values through the frontend — never ask them to paste secrets into chat. If you get a setup URL, share it in one short sentence and wait for the user's confirmation. Use `major vars set KEY=VALUE` only for known values the user explicitly wants you to configure.
- **New connector**: `mcp__plugin_major_major__request_resource_setup`; load `using-connectors` for that flow. An existing connector can be added to app code separately.
