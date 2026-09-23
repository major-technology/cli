---
name: skill-builder
description: Create and manage Major agent skills — versioned file bundles (SKILL.md + references + scripts) that an attached agent loads on demand — and the sandbox-based edit/validate/save/publish lifecycle.
---

# Building a Major agent skill

A _skill_ is a versioned bundle — `SKILL.md` (required) + optional `references/*.md` + optional `scripts/*.{js,ts}` — that an attached agent loads on demand. Turn the user's intent into a focused bundle. Pure-instruction skills are fine; when a skill does real work it's a **script** that talks to a resource through the proxy or a generated client (never hand-written client/auth code). You author the bundle on the skill's sandbox and edit it through the sandbox tools (the "Working with sandboxes" section of your system prompt covers addressing, provisioning, and sharing).

Orchestrator tools are `mcp__orchestrator-platform__*` (`list_editable_skills`, `mount`, `create_skill`, `publish`, `get_skill`). File editing and sync go through the sandbox tools `mcp__major__sandbox_*` (`sandbox_read_file`, `sandbox_edit_file`, `sandbox_write_file`, `sandbox_pull`, `sandbox_push`, `sandbox_validate`), each called with `skill: "<skillId>"` as the target. `publish` takes the same `skill` argument.

## The working files, saving, and publishing

Each skill's working copy lives on its own sandbox under the workspace root. Two separate steps take it off the sandbox:

- **Save** (`sandbox_push`) validates the bundle and writes it as a new immutable version. Nothing about what attached agents load changes — saving is free. **The sandbox is torn down once it goes idle and comes back seeded from the last saved version, so anything unsaved is lost.**
- **Publish** (`publish`) points the skill at its latest saved version. This is the only thing that changes what attached agents load.

So a saved-but-unpublished draft is inert to attached agents — the agent that has it attached is the one that loads `SKILL.md`, at the last **published** version. Scripts are different: run them on this sandbox before saving.

Always use a `skillId` returned by `list_editable_skills` or `create_skill` — never invent one. If this chat is pinned to a skill, the "Working with this skill" section of your system prompt carries the bound-chat rules (its id, recovery, discard) — those win.

## Save discipline

- **Never `sandbox_pull` routinely** — it overwrites the sandbox folder with the last saved version and destroys any unsaved edits, the user's included. Pull only to recover a corrupted folder or on the user's explicit ask to discard.
- **Always `sandbox_push` before you finish a turn in which you edited files.** Unsaved work dies with the sandbox. Saving needs no permission and changes nothing about what attached agents load.
- **Publish only when the user asks for it.** That is the moment attached agents start seeing your changes.

## Lifecycle

- **Edit existing**: `list_editable_skills` to find it, then `mount({skill: "<skillId>"})` — it mounts (or joins) the skill's sandbox. Edit the files with the sandbox tools, saving as you finish each round.
- **New**: `create_skill({})` — it creates an empty draft (server-minted `skillId`), mounts its sandbox seeded with a scaffold, and returns where the files live. Build the bundle in that folder.
- **Check a draft**: `sandbox_validate` — validates the sandbox bundle against the platform's rules without saving. Saving also validates; on failure nothing is saved and the error list comes back — fix the files and save again.
- **Test scripts**: `tsx` and `@major-tech/resource-client` are on the sandbox (same as the agent pod). From the workspace root, `tsx scripts/<task>.ts` via the `sandbox_bash` tool — env already has `MAJOR_GO_RESOURCE_URL` / `MAJOR_RESOURCES_API_TOKEN`. Extra `package.json` deps: `npm install` first (`node_modules` isn't saved). `sandbox_validate` only checks bundle shape (frontmatter, file rules), not script correctness.
- **Save**: `sandbox_push({notes})` — every save writes a new immutable version.
- **Publish**: `publish({skill})` — makes the latest saved version live. Name the target the same way the sandbox tools do: pass the `skillId` you passed to `sandbox_push`. Only on the user's explicit go-ahead. Add `versionId` to roll back to an earlier version.

## Bundle rules

- Frontmatter needs `name` (lowercase kebab-case — publishing claims it as the skill's slug) and `description`. Any file type is allowed, binaries included; `sandbox_validate` and every `sandbox_push` enforce the path and size rules with errors naming what to fix.
- The `description` is the catalog line the SDK injects so the model can decide whether to load the skill — write `[what it does] + [when to use it] + [key capabilities]`, third person, with user trigger phrases. Good: "Fetches and summarises Stripe charges and refunds. Use when the user asks to summarise charges, look up a refund, or mentions a Stripe customer id." Bad: "helper for Stripe" (vague) or "Fetch and summarise Stripe charges; expects a stripe customer id" (function signature, no triggers).
- Keep the body task-focused and link long detail to `references/*.md`.
- `package.json` (`type: module`) is scaffolded for you — never author it. Add `dependencies` to it for extra npm packages. Never write `node_modules` (excluded from the saved version).

## Scripts & resources

A script runs at agent-invocation time via `tsx <plugin-dir>/skills/<slug>/scripts/<task>.ts`. On this sandbox the same command is `tsx scripts/<task>.ts` from the workspace root. **Contract:** print one JSON line to stdout — `{ok:true,...}` or `{ok:false,error:"..."}` — and exit non-zero on failure.

**npm packages:** `@major-tech/resource-client` is preinstalled — just `import` it; don't add it to `package.json`. For any other package, add it to `package.json`.

**Proxyable resources (the default)** — anything `list_resources` flags `proxy.compatible: true`. Use `createProxyFetch` from the package ROOT (not `/next` — that needs a Next app); the proxy injects auth, so never set `Authorization`:

```js
import { createProxyFetch } from "@major-tech/resource-client";

const proxyFetch = createProxyFetch({
	baseUrl: process.env.MAJOR_GO_RESOURCE_URL,
	resourceId: "<id from list_resources>",
	majorJwtToken: process.env.MAJOR_RESOURCES_API_TOKEN,
});
// full upstream URL, or a relative path the proxy resolves against the resource's base
const res = await proxyFetch("https://api.hubapi.com/crm/v3/objects/contacts");
process.stdout.write(
	JSON.stringify(res.ok ? { ok: true, data: await res.json() } : { ok: false, error: `proxy ${res.status}` }),
);
```

Read `references/http-proxy.md` in the `using-connectors` skill for the full proxy reference.

**Non-proxyable resources** (databases, non-HTTP connectors) — generate a typed client; don't hand-write it. The `sandbox_add-resource-client({resourceId, resourceName, resourceType, resourceDescription, skill: "<skillId>"})` writes a `.ts` client into `clients/` and returns the import line — use it VERBATIM (it may end in `.ts`; never rewrite to `.js`). `sandbox_remove-resource-client` (same target) deletes one.

```ts
import { ordersDbClient } from "../clients/ordersDb.ts"; // use the import the tool returned, verbatim

// invoke(...) → { ok, result } | { ok: false, error: { message } }; args vary by resource family
const res = await ordersDbClient.invoke("SELECT count(*) AS n FROM orders", [], "weekly-orders");
process.stdout.write(
	JSON.stringify(res.ok ? { ok: true, count: res.result.rows[0].n } : { ok: false, error: res.error.message }),
);
```

## Style

Be terse — the Files tab mirrors the sandbox live, so don't echo file contents back. Ask about _intent_ (task, resources, inputs/outputs) when it's unclear, then write the files yourself rather than going file-by-file. Don't write or run unrelated application code — you're editing a file bundle, not building software.
