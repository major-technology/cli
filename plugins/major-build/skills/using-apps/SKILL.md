---
name: using-apps
description: Use for ANY call against a deployed Major app — listing apps the user can invoke, reading an app's skill (endpoints and request/response shapes), and issuing GET or write requests. Load before querying, calling, or probing a deployed app. Do not use this skill to create, edit, mount, or deploy an app — that is app-builder.
---

# Using deployed apps

A deployed Major app is compute you call over HTTP. Major handles auth and plumbing. This skill is for **invoking** an existing deployed app — not for creating, editing, mounting, or deploying one (load `app-builder` for that).

Always use an `applicationId` returned by `list_apps` with `include_read_only: true` — never invent one. That is the catalog of apps you can call; without `include_read_only`, `list_apps` shows only the apps the user can edit.

## Steps

1. **Identify the app.** `list_apps` with `include_read_only: true` shows the deployed apps you can call. Read `/workspace/memory/apps/<applicationId>/purpose.md` to understand what each one does. If more than one could match, ask which.
2. **Read its skill.** Then call `get_app_skill` with its id for its endpoints and request/response shapes — usually enough. Follow that skill; do not guess paths or payloads.
3. **Call it.** `do_get_request` for GET. `do_requests` for any non-GET (the user will be asked to approve writes).

Do not issue writes via `do_requests` just to learn a shape — `get_app_skill` first, then `do_get_request` only if something is still unclear, and ask the user before any write.

## Related

- To change the app itself (sandbox, preview, deploy), load `app-builder`.
- To grant an agent access to an app, load `agent-builder` — attach via `list_apps` (editable apps only).
- For a scheduled or event-driven call, load `workflow-builder` and use an `app_call` node.
