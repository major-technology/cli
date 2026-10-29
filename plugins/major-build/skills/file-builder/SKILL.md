---
name: file-builder
description: Create and edit Major Files — static pages (HTML plus CSS, JS, JSON, images, fonts, markdown) with their data written in, made in minutes with no backend or deploy — and the sandbox-based edit/validate/save/publish lifecycle. Use for anything static the user will read or share with colleagues, such as a report, briefing, explainer, or one-off chart, or to change an existing File. A page that needs fresh data, saves input, or runs compute is an app instead.
---

# Building a Major File

A _File_ is a versioned bundle of `.html` entries plus the assets they use (CSS, JS, JSON, images, fonts, markdown): at most 100 entries and 10 MB, relative paths only, and at least one `.html` entry. Each `.html` entry is a page users view, served on Major's isolated content domain with every File on its own origin; Major injects a small frame-helper script into HTML on save, so don't add one yourself. Files have no tab in the Major web app — you create, edit, and open them for the user. `index.html` is the page the user opens, so keep it as the entry and link any other pages from it. A File is private to the user who created it. You author the bundle on the File's sandbox and edit it through the sandbox tools (the "Working with sandboxes" section of your system prompt covers addressing, provisioning, and sharing).

Finding, creating, and opening Files is on `mcp__plugin_major_major__*` (`list` and `create` with `target_type: "file"`, `start_sandbox`). File editing goes through the sandbox tools `mcp__plugin_major_major__sandbox_*` (`sandbox_read_file`, `sandbox_edit_file`, `sandbox_write_file`, `sandbox_bash`), each called with `target_type: "file"` and `target_id: "<fileId>"` as the target. Sync and publishing are the `major` CLI, run through `sandbox_bash` in `/workspace/file`: `major pull`, `major push -m "<notes>"`, `major validate`, `major publish --yes`.

## The working files, saving, and publishing

The bundle lives directly at the top of `/workspace/file`. Two separate steps take it off the sandbox:

- **Save** (`major push`) validates the bundle and writes it as a new immutable version. Viewers still see the previous published version. **The sandbox is torn down once it goes idle and comes back seeded from the last saved version, so anything unsaved is lost.**
- **Publish** (`major publish --yes`) makes the latest saved version what viewers see.

Unlike an agent, skill, or workflow, a File does nothing until it is viewed, so an unpublished File is useless: when the user asked for the File, publish after you push — unless the user said not to.

Always use a `fileId` returned by `list` or `create` (`target_type: "file"`) — never invent one. Pepper's priority Files are not Files you edit here: they don't appear in `list` and can't be opened in a file sandbox.

## Save discipline

- **Never `major pull` routinely** — it overwrites the sandbox folder with the last saved version and destroys any unsaved edits, the user's included. Pull only to recover a corrupted folder or on the user's explicit ask to discard.
- **Always `major push` before you finish a turn in which you edited files**, then `major publish --yes` so the user sees the change.

## Lifecycle

- **New**: `create({target_type: "file", name, description?})` — it creates the File with a starter v1 (`index.html`, an empty page) and starts its sandbox. Replace `index.html` with the real page.
- **Edit existing**: `list({target_type: "file"})` returns `fileId, name, description, isPublished, updatedAt, canEdit`; then `start_sandbox({target_type: "file", target_id: "<fileId>"})` mounts (or joins) its sandbox.
- **Check**: `major validate` — checks the bundle (an `.html` entry, entry and size caps, safe paths) without saving. Saving also validates; on failure nothing is saved and the error list comes back — fix the files and save again.
- **Save**: `major push -m "<notes>"`. **Publish**: `major publish --yes`.

## Writing the page

The sandbox is lightweight: no preview server, no browser. You cannot render the page, so make it correct by construction:

- Self-contained HTML: inline `<style>`/`<script>`, or relative paths to CSS/JS/JSON in the bundle. Scripts from other origins load only from cdnjs.cloudflare.com or cdn.jsdelivr.net, at pinned versions; the page can fetch nothing but its own files.
- Embed the data the page shows, inline or as a bundled `.json` fetched by relative path; query the user's resources first and write the results in.
- Every relative reference must point at a file in the bundle.

## Style

Be terse — don't echo file contents back into chat. Ask about _intent_ (what the page is for, what data it shows) when it's unclear, then write the files yourself.
