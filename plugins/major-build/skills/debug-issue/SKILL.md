---
name: debug-issue
description: Use this skill when the user asks to debug, investigate, inspect for bugs, fix an issue, or says the app is broken, failing, wrong, blank, errored, or not working. Also use it when you observe a concrete existing failure yourself, including runtime errors, failed tool/API calls, browser console errors, app errors, app logs, actual visual regressions, failing lint/tests/builds, deployment issues, bad data, auth issues, or flaky behavior. Do not use this skill for forward-looking UX polish requests like "make the UX nicer" or "improve the design" unless the user identifies a specific existing broken behavior or regression to verify.
---

You are debugging an application issue. Treat debugging as evidence gathering followed by the smallest responsible fix.

## Trigger boundary

This skill is retrospective: use it to investigate existing behavior, confirmed failures, or regressions.

Do not turn subjective or forward-looking product requests into debugging work. If the user asks for UX to be "nice", "better", "cleaner", "polished", or similar without naming a broken current behavior, treat it as design/implementation work instead of invoking this debug workflow.

Use this skill for UX only when there is something concrete to reproduce, such as:

- "The button is clipped on mobile."
- "This used to work and now the layout is broken."
- "Clicking Save does nothing."
- "The modal looks wrong after the latest change."

Start by clarifying the failure mode when needed:

- What did the user expect?
- What actually happened?
- Where does it happen: local preview, deployed app, a specific route, an API endpoint, a background job, a resource connector, or all of the above?
- Is the issue reproducible, intermittent, or based on a user report?

If the user asks a vague follow-up like "debug the issue", "fix the bug", or "find what's wrong" in a session with a running preview, do not ask for more details first. Treat the current app as the reproduction target, inspect `http://localhost:3000`, and gather browser evidence before deciding whether you need clarification.

Prefer direct evidence over guesses. Use the most relevant tools below.

## Preview and browser issues

The app's preview is always served at `http://localhost:3000`. There is no other port or host to discover — navigate, screenshot, and read logs against that URL every time.

Always check app errors and app logs before opening Playwright. `major app errors list` and `major app logs` explain almost every server, route handler, and runtime failure without needing the browser. Run these CLI commands through `mcp__major__bash` in the mounted app workspace; load `app-builder` if you need to mount it first. Only open the browser when the bug is purely visual, layout-related, or only observable from the rendered page.

When you do use Playwright, you are limited to **looking at the page**, not driving it:

- ✅ `mcp__major__browser_navigate` — open a URL on `http://localhost:3000`.
- ✅ `mcp__major__browser_take_screenshot` — capture the rendered page. Always save under `/workspace/.session-files/`; never use `/workspace/app`, repo paths, or relative paths.
- ✅ `mcp__major__browser_snapshot` — accessibility snapshot for reading text/structure.
- ✅ `mcp__major__browser_wait_for` — wait briefly for a pending/loading state to settle before re-screenshotting.
- ✅ `mcp__major__browser_console_messages` — read the browser console when investigating client-side errors.

Do **not** click, type, drag, hover, resize, fill forms, press keys, evaluate JavaScript, or otherwise interact with or mutate the page. If the bug only reproduces through user interaction, describe the reproduction steps and ask the user to perform them — do not attempt to drive the page yourself.

- Capture the page state before editing code when the problem is visual or route-specific.
- If a screenshot path is produced, inspect it yourself before making conclusions.
- For challenge-style prompts where the app intentionally contains a hidden bug, exercise the primary UI flow in the browser and look for mismatches between expected behavior and rendered behavior.

When investigating a user-reported issue whose affected surface is broad — a shared component, layout, theme/CSS, or anything imported by many pages — discover the app's routes from `http://localhost:3000/__major_devtools_routes__.json` and check the affected ones in the browser rather than guessing from code alone. For a single-page complaint, navigate and screenshot that page.

## App errors

For blank pages, error overlays, server crashes, failed route handlers, or deployed runtime failures, check app errors early.

- Use `major app errors list` to find recent errors.
- Use `major app errors get <errorId>` for details before editing.
- Prefer sourcemapped stack traces and request context from app errors over broad log searches.
- After fixing and committing a confirmed app error, use `major app errors resolve <errorId>` only when the issue is actually addressed.
- If app errors are unavailable and the task is specifically about runtime error monitoring, add the reporter scaffolding before running `major app errors enable`.

## App logs

Use logs when behavior depends on server startup, route handlers, background work, request handling, or errors that are not captured by app errors.

- For the live local preview/dev server, run `major app logs --preview`.
- For deployed app/runtime logs, run `major app logs`.
- Start with a small `--limit` and a focused `--search` term from the route, error message, request ID, resource name, or timestamp. Search is case-sensitive for deployed logs and case-insensitive with `--preview`.
- Use `--since` and `--until` for time-bounded reports.
- Both deployed and preview logs support pagination: use `--json` to obtain `nextToken`, then `--next-token <token>` after reading the first result. Keep `--preview` when paging preview logs.
- Logs are reverse chronological. Do not assume absence of evidence means the code path did not run if the time window or search term is too narrow.

## Code and data investigation

After collecting runtime evidence, trace the responsible code path.

- Search for the route, component, handler, resource client, env var, or error text using `mcp__major__grep` / `mcp__major__glob`.
- Read nearby code with `mcp__major__read_file` before editing with `mcp__major__edit_file` or `mcp__major__write_file`.
- Follow the app-specific conventions provided to you as context.
- If data shape is involved, inspect the relevant MCP resource or generated client before changing UI assumptions.
- If an external API, connector, auth provider, or resource appears unavailable, verify that dependency before patching app code. Report dependency failures as uncertainty instead of looping on frontend fixes.
- If auth or environment is involved, verify environment variables and resource setup rather than hard-coding fallbacks.

## Fix and verify

Make the smallest fix that explains the evidence.

- Re-run the exact reproduction path after editing.
- If patching shared or reusable code, verify other consumers of that code path still work.
- For frontend fixes, re-check screenshots and browser console output.
- For server or deployed-runtime fixes, re-check app errors and focused app logs when applicable.
- Run lint before finishing.
- Do not run full build commands unless the issue is specifically a build failure or lint cannot exercise the failure.

When reporting back, include:

- The confirmed cause.
- The files changed.
- The verification performed.
- Any remaining uncertainty, especially if logs/errors were unavailable or the issue was intermittent.
