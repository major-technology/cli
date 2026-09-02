---
name: using-github-connector
description: Implements GitHub repository, issue, pull request, release, content, branch, and authenticated git operations using the GitHub connector, mounted MCP tools, generated clients, and HTTP proxy. Use when doing ANYTHING that touches GitHub in any way, including cloning a private repository.
---

# Major Platform Resource: GitHub

## Setting Up a GitHub Connector

GitHub uses a GitHub App installation. When the user asks to connect GitHub:

1. Call `mcp__resource-setup__request-resource-setup` with `subtype: "github"`.
2. Ask the user to finish the GitHub installation flow and select the repositories the app may access.
3. After setup completes, call `mcp__resources__list_resources` and use the connected GitHub resource's `resourceId` and mounted MCP slug.

---

## Common: Interacting with Resources

**Security:** Never connect to GitHub with credentials placed in source code. Use the connector's MCP tools, generated client, or HTTP proxy. Only request a raw token for git operations such as clone, fetch, or push. GitHub installation tokens are short-lived secrets: never print, log, commit, or persist them.

**Description field:** Include a short `description` (~5 words) in resource MCP calls that accept it, such as `"Get token to clone repo"`.

**Four ways to interact with GitHub:**

1. **Mounted GitHub MCP tools** (direct, preferred): The connected GitHub MCP server exposes tools as `mcp__<slug>__<toolName>`. Use `mcp__resources__list_resources` to discover the resource and its slug. The hosted tool catalog covers repositories, files, issues, pull requests, branches, commits, and releases.
2. **Git token tool** (git CLI only): Call `mcp__resources__github_get_git_token` with the GitHub `resourceId`. Optionally downscope it to repository names and permissions.
3. **Generated TypeScript client** (app code): Call `mcp__resource-tools__add-resource-client` with the `resourceId`. The generated client is created in `/clients/` (Next.js) or `/src/clients/` (Vite).
4. **HTTP proxy** (Next.js app code or direct MCP calls): Use `createProxyFetch` from `@major-tech/resource-client/next`, or `mcp__resources__http_proxy_get` / `mcp__resources__http_proxy_invoke`, for GitHub REST or GraphQL endpoints not covered by a mounted MCP tool. See [using-http-proxy](../http-proxy/SKILL.md).

**Do not guess tool names or argument shapes.** Mounted tools come from GitHub's hosted MCP server and may change. Inspect the tools available under the connector's actual slug before calling them. After generating a TypeScript client, read its source to verify exact methods and signatures.

**Prefer least privilege:** For clone/fetch, request only the target repository with `permissions: { "contents": "read" }`. For push, use `contents: "write"` only when required.

**Framework note:** In Next.js, generated resource clients and `createProxyFetch` must be used in server-side code only. Never expose a GitHub token or Major JWT to a Client Component or browser.

**Error handling:** Always check `result.ok` before accessing `result.result` from a generated client.

**Invocation keys must be static strings** such as `"clone-project-repo"`, never dynamic values such as `` `${owner}-${repo}` ``.

---

## MCP Tools

### Mounted GitHub tools

Mounted tools use `mcp__<slug>__<toolName>`. Discover the exact catalog and schemas from the connected server. Typical operations include:

- Reading or updating repository contents
- Listing, creating, and updating issues
- Listing, creating, reviewing, and merging pull requests
- Creating branches and inspecting commits
- Listing and managing releases

Prefer these tools over minting a raw token when the requested operation is available through MCP.

### Git token tool

- `mcp__resources__github_get_git_token` — Mint a short-lived GitHub App installation token for git clone/fetch/push.

Arguments:

- `description`: brief user-visible operation label
- `resourceId`: UUID of the connected GitHub resource
- `repositories?`: repository **names** to scope the token to, such as `["my-repo"]`
- `permissions?`: permission subset, such as `{ "contents": "read" }`

The result includes `token`, `expiresIn`, and may include `gitHost`. Use `gitHost` when present instead of assuming `github.com`, because the connector may target GitHub Enterprise Cloud.

---

## Clone a Repository with a GitHub Token

**The correct clone flow is to mint a GitHub token first.** Do not try an anonymous clone for a private repository, do not use SSH credentials, and do not put the token directly in the clone URL because Git saves that URL in `.git/config`.

1. Find the GitHub resource with `mcp__resources__list_resources`.
2. Call `mcp__resources__github_get_git_token`:

```json
{
  "description": "Get token to clone repo",
  "resourceId": "<github-resource-id>",
  "repositories": ["example-repo"],
  "permissions": { "contents": "read" }
}
```

3. Put the returned token in a temporary environment variable and use a temporary `GIT_ASKPASS` helper. Use the returned `gitHost`, or `github.com` when it is absent:

```bash
export GITHUB_TOKEN='<token returned by github_get_git_token>'
export GITHUB_HOST='<gitHost returned by the tool, or github.com>'

askpass="$(mktemp)"
chmod 700 "$askpass"
cat >"$askpass" <<'EOF'
#!/bin/sh
case "$1" in
  *Username*) printf '%s\n' 'x-access-token' ;;
  *Password*) printf '%s\n' "$GITHUB_TOKEN" ;;
esac
EOF

GIT_ASKPASS="$askpass" GIT_TERMINAL_PROMPT=0 \
  git clone "https://${GITHUB_HOST}/example-org/example-repo.git"

rm -f "$askpass"
unset GITHUB_TOKEN GITHUB_HOST
```

Use a shell cleanup trap when scripting so the helper and token are removed on failure too. Never echo the token, include it in command output, or construct a remote like `https://x-access-token:<token>@github.com/...`; that form can leak through logs, process arguments, shell history, and `.git/config`.

If clone returns `Repository not found`, verify that the installed GitHub App can access the repository and that `repositories` contains the repository name (not `owner/name`). Mint a fresh token if it expired.

---

## TypeScript Client

```typescript
import { githubClient } from "./clients";

const result = await githubClient.getGitToken("clone-project-repo", {
  repositories: ["example-repo"],
  permissions: { contents: "read" },
});

if (!result.ok) {
  throw new Error(result.error.message);
}

// Keep this server-side and in memory only. Do not return or log the token.
const { token, expiresIn } = result.result;
```

Use the raw token only when launching a server-side git operation. For normal GitHub API work, prefer mounted MCP tools or the HTTP proxy so authentication remains injected server-side.

---

## HTTP Proxy

GitHub REST uses `https://api.github.com`; GraphQL uses `https://api.github.com/graphql`. The proxy injects the installation token, so **do not set `Authorization` yourself**.

```typescript
import { createProxyFetch } from "@major-tech/resource-client/next";

const githubFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.GITHUB_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

const response = await githubFetch("https://api.github.com/repos/example-org/example-repo/issues", {
  headers: {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
  },
});
```

For an enterprise connector, use the base URLs advertised by `mcp__resources__list_resources`; do not hard-code `api.github.com`.

## Tips

- GitHub App access is limited to repositories selected during installation and permissions granted to the app.
- Installation tokens generally expire in about one hour; rely on `expiresIn` and mint a new token rather than reusing an expired one.
- A repository selection error is not fixed by requesting broader token permissions; the user must grant the GitHub App access to that repository.
- Use `contents: "read"` for clone/fetch and `contents: "write"` only for push.
- Use mounted MCP tools for API operations so raw credentials stay out of model-authored application code.

**Docs:** [GitHub MCP server](https://github.com/github/github-mcp-server) · [GitHub REST API](https://docs.github.com/en/rest) · [GitHub App installation tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)
