---
name: agent-builder
description: Create and configure a Major agent — a versioned two-file bundle (agent.jsonc + prompt.md) edited on a sandbox — what each field means, how to research connectors and applications before writing the prompt, and the edit/validate/save/publish lifecycle.
---

# Building a Major agent

An _agent_ in Major is a saved AI configuration users can invoke. It is a **versioned two-file bundle** you author on the agent's own sandbox:

```
agent.jsonc   name, description, model, skills, env keys, connectors + applications
              (with their per-tool permission decisions nested inline)
prompt.md     the system prompt — the file IS the prompt, no wrapper
```

If you don't have enough information to write a good system prompt or pick connectors, ask the user — it is better to ask than to guess.

Orchestrator tools are `mcp__orchestrator-platform__*` (`list_editable_agents`, `mount`, `create_agent`, `get_agent`, `publish`). File editing and sync go through the sandbox tools `mcp__major__*` (`read_file`, `edit_file`, `write_file`, `pull`, `push`, `validate`), each called with `agent: "<agentId>"` as the target. `publish` takes the same `agent` argument.

## The working files, saving, and publishing

The working copy lives on the agent's sandbox under the workspace root. Two separate steps take it off the sandbox:

- **Save** (`push`) validates the bundle and writes it as a new immutable version. Nothing the agent runs changes — saving is free. **The sandbox is torn down once it goes idle and comes back seeded from the last saved version, so anything unsaved is lost.**
- **Publish** (`publish`) points the agent at its latest saved version. This makes the agent live.

An agent with no published version can't be run deployed at all — starting a session against it fails with "no published version yet". So a brand-new agent needs one `publish` before anyone can use it.

Always use an `agentId` returned by `list_editable_agents` or `create_agent` — never invent one. If this chat is pinned to an agent, the "Working with this agent" section of your system prompt carries the bound-chat rules (omit ids to target it).

## Save discipline

- **Never `pull` routinely** — it overwrites the sandbox files with the last saved version and destroys any unsaved edits, the user's included. Pull only to recover corrupted files or on the user's explicit ask to discard.
- **Always `push` before you finish a turn in which you edited files.** Unsaved work dies with the sandbox. Saving needs no permission and changes nothing about what the agent runs.
- **Publish only when the user asks for it.** That is the moment the agent's behaviour changes for everyone.

## Lifecycle

- **Edit existing**: `list_editable_agents` to find it, then `mount({agent: "<agentId>"})` — it mounts (or joins) the agent's sandbox. Edit the two files with the sandbox tools, saving as you finish each round.
- **New**: `create_agent({name, description})` — creates the agent (server-minted `agentId`), mounts its sandbox seeded with a scaffold bundle, and returns where the files live.
- **Check a draft**: `validate` — parses `agent.jsonc` against the schema without saving. Saving validates too (and additionally checks that every referenced skill/connector/app exists in the org); on failure nothing is saved and the error list comes back.
- **Save**: `push({notes})` — every save writes a new immutable version.
- **Publish**: `publish({agent})` — makes the latest saved version live. Name the target the same way the sandbox tools do: pass the `agentId` you passed to `push`. Only on the user's explicit go-ahead. Add `versionId` to roll back to an earlier version.

## `agent.jsonc`

The definition shape — fields, the allowed model ids, permission decisions — is the JSON Schema the Major API serves at `GET https://api.prod.major.build/public/agent.schema.json`, the single source of truth. YOU MUST CURL THIS SCHEMA BEFORE WRITING `agent.jsonc`; its `x-validatorRules` carry the rules beyond shape (membership grants access, ids must exist in the org, bundle is exactly the two files). `validate` and every `push` enforce all of it, with errors naming the offending path.

## Tool permissions

**Sensible defaults are already applied — usually don't touch this.** Read-only tools and `GET` endpoints default to `always_allow`; writes and every non-`GET` method default to `ask`. Only list a tool or endpoint explicitly when the user wants to deviate (e.g. "never let it delete anything").

To see the current picture: `list_resource_permissions({resourceId})` / `list_app_permissions({applicationId})` return every tool/endpoint with its decision **as of the published version**. To change one, edit that connector's `tools` (or that app's `endpoints`) array in `agent.jsonc`, then push and publish — there is no live permission-editing tool.

## Env variables

The bundle declares which env **keys** a version wants. A non-secret value can sit right next to its key in `agent.jsonc`. A key set to `null` has no value in the bundle, and the user supplies one.

**You never set a value, and you never see one.** Add the key to `env` with `null`, push, and tell the user to fill it in — the env section of the agent panel, on the right, lists every declared key with a value box. The value is stored encrypted per `(agent, key)` and shared across versions, so it survives every save, publish and rollback.

`get_agent` returns `envKeys` — every declared key with `hasValue`. Check it before you publish, and name the keys that are still empty.

There is no tool that sets an agent's env value.

On Slack there is no panel. Tell the user to open the agent in the web app to fill in a value.

## Picking connectors and applications

- Use `mcp__major__execute_resource_tool` with `toolName: "mcp__resources__list_resources"` to list the org's connectors; `mcp__orchestrator-platform__list_edit_apps` lists attachable apps. **Use `list_edit_apps`, not `list_use_apps`** — an agent can only be granted apps the user can edit.
- If no existing connector matches, call `mcp__interactions__request_resource_setup` to prompt the user to create one inline. `connectorId` is required — pass one you already know (e.g. `"postgresql"`, `"snowflake"`) or use `mcp__major__execute_resource_tool` with `toolName: "mcp__resources__search_connector_types"` to discover the connectors you can set up; ask if unsure. The tool blocks until the user finishes or declines; on success add the returned `resourceId` to `connectors` in `agent.jsonc`.
- Slack is provisioned automatically when the user installs the Major Slack integration (Settings → Integrations) and is intentionally not a creatable connector — if it's missing from `list_resources`, tell them to install the integration.
- If an existing connector needs more configuration to be usable (e.g. selecting a Google Sheets spreadsheet), call `mcp__interactions__request_resource_update` with the `resourceId` and what's missing.
- After adding skills, call `list_suggested_connectors` — it returns connectors the attached skills' scripts actually use that the agent can't access yet. Propose them to the user and add accepted ones to `connectors`; a skill whose connector is missing will fail at runtime. Entries with `canAdd=false` need access the current user doesn't have — tell them to ask an admin.
- Don't add connectors or applications speculatively — every one expands the agent's permissions. Keep the set minimal.

## Research before writing the prompt

A good system prompt names the actual tables, endpoints, and fields the agent will use — not "query the database". Probe what you attached before writing:

- **Connectors:** pass the matching canonical `mcp__resources__*` tool name to `mcp__major__execute_resource_tool` — `information_schema` + a few sample rows for SQL databases, object/property lists for CRMs, bucket/key listings for S3, an introspection or health call for APIs. Canonical resource tools are execution targets, not directly callable tools. Stop once you can write a confident prompt — you're not building a data dictionary.
- **Applications:** call `get_app_skill({applicationId})` first (it returns the endpoints and request/response shapes — usually enough). Probe live endpoints with `do_get_request` only if something is still unclear, and never issue writes via `do_requests` just to learn a shape — ask the user first.

Then cite what you found in `prompt.md`: "query `analytics.daily_sessions` filtered by `user_id`", not "ask the database about sessions". A good prompt is 5–20 lines — if the user gives you a one-liner, draft a proper prompt yourself, after the research, not before.

## Attaching skills

Skills are reusable instruction bundles authored in the Skill Library; attached skills auto-load when the agent's sessions start. `list_attachable_skills` returns `{id, slug, description}`; add the ids you want to the `skills` array in `agent.jsonc`. Attach only skills whose `description` clearly fits the agent's job — a skill the model never uses is noise. If no fitting skill exists and the user has described one, use the `skill-builder` skill to build it.

## Running on a schedule

An agent has **no schedule of its own** — running an agent on a cadence is a property of a _workflow_ whose trigger fires the agent. If the user wants this agent to run on a schedule, load the `workflow-builder` skill and build a workflow there.

## Connecting the agent to Slack

An agent can get its own Slack bot so people @mention it in their workspace: `connect_agent_to_slack({agentId})` provisions a dedicated Slack app and returns the install URL for the user to open. `pause_agent_channel` / `resume_agent_channel` silence and re-enable it. `delete_agent_channel` is permanent — reconnecting later creates a brand-new bot identity that won't reattach to existing threads, so only on the user's explicit ask.
