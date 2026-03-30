# CLI MCP Servers — Implementation Plan

**Date:** 2026-03-30
**Goal:** Add org-level MCP endpoints that the CLI can use with its existing auth token, ship them in the Claude Code plugin's `.mcp.json`, and auto-approve read-only tools.

---

## Context

### Current State

**go-api resource MCP** (`/internal/apps/v1/:applicationId/mcp`)
- 92 tools across 24 resource types
- Auth: `x-major-jwt` (app-scoped development JWT)
- ApplicationId required in URL path
- Only 3 tools actually use applicationId: `setup_managed_database`, `majorauth_share_access`, `majorauth_revoke_access`
- All other tools work purely with org + resource context

**apps/api platform MCP** (`/mcp/:applicationId`)
- 12 tools: `get_app_status`, 7 memory tools, 2 AI proxy tools
- Auth: `x-major-jwt` with `type: "development"`
- ApplicationId required in URL path

**CLI auth model:**
- User runs `major user login` → device flow → gets Bearer token (type `"cli"`) stored in keychain
- Default org stored separately in keychain via `token.GetDefaultOrg()`
- CLI sends `Authorization: Bearer <token>` + `organizationId` in request body
- Node API validates via `tokenAuthMiddleware.assertCliToken()` → extracts userId/email from token
- **This is different from** the session-based auth used by `/internal/authorize` (cookie-based, `auth.api.getSession()`)

### What We're Building

New **org-level** MCP endpoints that:
- Accept the CLI's existing Bearer token (`Authorization: Bearer <token>`)
- Don't require applicationId in the URL
- Tools that need an applicationId accept it as a **tool input parameter**
- The plugin ships `.mcp.json` with `headersHelper` — no per-project setup needed

---

## Plan

### Step 1: New CLI Auth Endpoint on Node API

**File:** `apps/api/src/api/routes/internal/cli-authorize.ts` (new)
**Route:** `POST /internal/cli-authorize`

A new authorization endpoint specifically for CLI token auth, mirroring `/internal/authorize` but using `tokenAuthMiddleware` instead of session-based auth.

```typescript
// Request
{
  permission: string,         // e.g. "resource:build"
  resourceId?: string,        // for resource-level checks
  applicationId?: string,     // for app-level checks
  organizationId: string      // CLI's default org
}

// Response
{
  authorized: boolean,
  userId: string,
  organizationId: string,
  error?: string
}
```

**How it works:**
1. `tokenAuthMiddleware.assertCliToken()` validates the Bearer token, extracts userId
2. Verify user is a member of the provided `organizationId`
3. Check permission using `authzDAO` (same logic as `/internal/authorize` but org comes from request body, not session)
4. Return userId + organizationId

**Why a new endpoint:** The existing `/internal/authorize` uses `auth.api.getSession()` which resolves org from the session cookie's `activeOrganizationId`. CLI tokens don't have sessions — the org comes from the keychain and must be sent explicitly.

---

### Step 2: New CLI Token Command

**File:** `cli/cmd/user/token.go`

```
major user token
```

Prints the CLI's stored Bearer token to stdout. Hidden command (internal/tooling use).

- Reads token from keychain via `token.GetToken()`
- Prints raw token (no formatting, no newline decoration)
- Used by the plugin's `headersHelper` script

**No app-id needed** — token is user-scoped, org comes from default org.

---

### Step 3: New Org-Level MCP Endpoint on Go API

**File:** `go-api/server/ep_mcp_cli.go` (new)
**Route:** `POST /cli/v1/mcp` (no applicationId in path)

A new org-level MCP endpoint that authenticates via CLI Bearer tokens.

**New middleware:** `CheckCliAuth()` in `go-api/server/middleware.go`

```go
func (s *Server) CheckCliAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Read organizationId from header
        organizationID := c.GetHeader("x-major-org-id")
        if organizationID == "" {
            c.AbortWithStatusJSON(400, errors.ErrorBadBody)
            return
        }

        // Forward Authorization header to Node API's new /internal/cli-authorize
        result, err := s.authClient.CliAuthorize(c.Request.Header, organizationID)
        if err != nil || !result.Authorized {
            c.AbortWithStatusJSON(401, errors.ErrorUnauthorized)
            return
        }

        c.Set(constants.ContextKeyOrganizationID, result.OrganizationID)
        c.Set(constants.ContextKeyUserID, result.UserID)
        c.Next()
    }
}
```

**New auth client method:** `CliAuthorize()` in `go-api/clients/auth/client.go`
- Calls `POST /internal/cli-authorize` on Node API
- Forwards `Authorization: Bearer <token>` header
- Sends `organizationId` in body
- No permission check needed at the gate — individual tool calls check resource permissions

**Handler:** `handleCliMCPRequest()` in `ep_mcp_cli.go`
- Same as `handleMCPRequest()` but:
  - `ApplicationID` is empty string (not from URL)
  - `MajorJWT` is empty (not used)
  - `GetResourceConfig` skips app-scoped environment resolution — uses org default environment
  - `ListResources` lists all org resources (no app filtering)
  - `SetupManagedDatabase` passes applicationId from tool input (if provided)

**Environment resolution change in `getResourceConfigForMCP`:**
- When `applicationID` is empty, skip `GetUserEnvironmentChoice` (which is per-app)
- Fall straight through to default build environment
- This is already the fallback behavior (lines 240-250 of ep_mcp.go)

**`listResourcesForMCP` change:**
- When `applicationID` is empty, skip `AppResourcePermissions` call (which requires app context)
- Instead, list all org resources the user has `resource:build` permission on
- New method: `CliResourcePermissions()` on the auth client that checks resource-level permissions without app context

**Tools that need applicationId:**
- `setup_managed_database` — accepts optional `applicationId` in tool args. If not provided, returns error saying applicationId is required for this tool.
- `majorauth_share_access` / `majorauth_revoke_access` — same: applicationId required as tool input, error if missing.
- All other tools work unchanged.

**Routes:**
```go
func (s *Server) addCliMCPRoutes(router *response.Router) {
    cliMCPRoutes := router.Group("/cli/v1/mcp", s.CheckCliAuth())
    cliMCPRoutes.GET("/tools", s.handleGetMCPToolsMetadata) // reuse existing
    cliMCPRoutes.RawAny("", s.handleCliMCPRequest)
}
```

---

### Step 4: New Org-Level Platform MCP Endpoint on Node API

**File:** `apps/api/src/api/routes/mcp/major-platform-cli.ts` (new)
**Route:** `POST /mcp/cli`

Same as existing `/mcp/:applicationId` but:
- Auth: `tokenAuthMiddleware.assertCliToken()` instead of `appAuthMiddleware.verifyDevAppHandler()`
- Org from request header `x-major-org-id`
- `applicationId` not required — tools that need it accept it as input
- `get_app_status` tool — accepts applicationId as tool input parameter
- Memory tools — work at org level (already org-scoped in S3 paths: `memory/organization/...`)
- AI proxy tools — accept applicationId as tool input

---

### Step 5: CLI Command to Get Org ID

**File:** `cli/cmd/org/id.go`

```
major org id
```

Prints the default org ID to stdout. Hidden command.
- Reads from keychain via `token.GetDefaultOrg()`
- Used by plugin `headersHelper` script for the `x-major-org-id` header

---

### Step 6: Plugin `.mcp.json` with `headersHelper`

**File:** `cli/claude-code-plugin/.mcp.json`

Uses `headersHelper` ([docs](https://code.claude.com/docs/en/mcp#use-dynamic-headers-for-custom-authentication)) to dynamically resolve auth headers at connection time. Claude Code runs the helper command, which must output a JSON object of header key-value pairs to stdout.

```json
{
  "mcpServers": {
    "major-resources": {
      "type": "http",
      "url": "https://resource-api.major.build/cli/v1/mcp",
      "headersHelper": "${CLAUDE_PLUGIN_ROOT}/scripts/get-headers.sh"
    },
    "major-platform": {
      "type": "http",
      "url": "https://api.major.build/mcp/cli",
      "headersHelper": "${CLAUDE_PLUGIN_ROOT}/scripts/get-headers.sh"
    }
  }
}
```

**File:** `cli/claude-code-plugin/scripts/get-headers.sh`

```bash
#!/bin/bash
TOKEN=$(major user token 2>/dev/null)
ORG=$(major org id 2>/dev/null)

if [ -z "$TOKEN" ] || [ -z "$ORG" ]; then
  echo '{"x-major-error": "Not authenticated. Run: major user login"}' >&2
  exit 1
fi

echo "{\"Authorization\": \"Bearer $TOKEN\", \"x-major-org-id\": \"$ORG\"}"
```

**How `headersHelper` works:**
- Executes at connection time (session start + reconnects)
- Must output a JSON object of string key-value pairs to stdout
- 10-second timeout per execution
- Has access to user's PATH (can call `major` CLI)
- Receives `CLAUDE_CODE_MCP_SERVER_NAME` and `CLAUDE_CODE_MCP_SERVER_URL` env vars
- `${CLAUDE_PLUGIN_ROOT}` resolves to the plugin's install directory

**Plugin `.mcp.json` notes:**
- Plugin MCP servers "work identically to user-configured servers" per the docs
- Plugin MCP servers start automatically when the plugin is enabled
- `${CLAUDE_PLUGIN_ROOT}` is supported for referencing bundled files

---

### Step 7: Auto-Approve Read-Only Tools

PreToolUse hooks receive `tool_name` and `tool_input` via stdin JSON, but **do not** receive MCP tool annotations like `readOnlyHint`. Two approaches:

**Approach A: PreToolUse hook + CLI command (recommended)**

**File:** `cli/claude-code-plugin/hooks/hooks.json`

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "mcp__major_resources__*",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/scripts/check-readonly.sh"
          }
        ]
      }
    ]
  }
}
```

**File:** `cli/claude-code-plugin/scripts/check-readonly.sh`

Reads `tool_name` from stdin JSON, calls `major mcp check-readonly <tool_name>`.

**File:** `cli/cmd/mcp/check_readonly.go`

Hidden command that checks if an MCP tool is read-only:
1. Hits `GET /cli/v1/mcp/tools` to get tool metadata (with `readOnly` flag)
2. Caches response locally (`~/.major/cache/tool-metadata.json`, TTL: 24 hours)
3. Outputs `{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "allow"}}` if read-only
4. Outputs nothing (or exits normally) if not read-only, letting Claude Code prompt the user

**Approach B: Static permission rules (simpler fallback)**

If hooks prove unreliable, use `settings.json` in the plugin with allow rules for known read-only tool name patterns:

```json
{
  "permissions": {
    "allow": [
      "mcp__major_resources__list_resources",
      "mcp__major_resources__list_resource_context",
      "mcp__major_resources__postgresql_psql",
      "mcp__major_resources__*_query",
      "mcp__major_resources__*_list_*",
      "mcp__major_resources__*_get_*",
      "mcp__major_resources__*_describe*"
    ]
  }
}
```

This is static and needs manual updates when new read-only tools are added, but requires no CLI command or caching.

**Recommendation:** Start with Approach A. Fall back to B if hook performance or reliability is an issue.

**Update `plugin.json`** to reference hooks:
```json
{
  "hooks": "./hooks/hooks.json"
}
```

---

### Step 8: Update Existing `GenerateMcpConfig()`

**File:** `cli/utils/mcp.go`

Stop generating the old app-scoped MCP config in `.mcp.json`. The plugin's org-level MCP via `headersHelper` is strictly better — it works without being in a project directory and doesn't need token refresh.

- Remove the `major` entry from `GenerateMcpConfig()` output
- Keep the function for backward compat but have it generate an empty `.mcp.json` (or remove the call sites in `clone`/`create`/`start`)
- Users who need the old app-scoped flow can use the existing endpoints directly

---

## Summary of Changes

### New Files

| File | Repo | Purpose |
|------|------|---------|
| `apps/api/src/api/routes/internal/cli-authorize.ts` | mono-builder | CLI token auth endpoint |
| `apps/api/src/api/routes/mcp/major-platform-cli.ts` | mono-builder | Org-level platform MCP |
| `go-api/server/ep_mcp_cli.go` | mono-builder | Org-level resource MCP endpoint |
| `cli/cmd/user/token.go` | cli | Print CLI token to stdout |
| `cli/cmd/org/id.go` | cli | Print default org ID to stdout |
| `cli/cmd/mcp/mcp.go` | cli | MCP command group |
| `cli/cmd/mcp/check_readonly.go` | cli | Read-only tool check for hooks |
| `cli/claude-code-plugin/.mcp.json` | cli | Plugin MCP config with `headersHelper` |
| `cli/claude-code-plugin/scripts/get-headers.sh` | cli | Dynamic auth header script |
| `cli/claude-code-plugin/scripts/check-readonly.sh` | cli | Hook script for read-only check |
| `cli/claude-code-plugin/hooks/hooks.json` | cli | Auto-approve hooks |

### Modified Files

| File | Change |
|------|--------|
| `go-api/server/middleware.go` | Add `CheckCliAuth()` |
| `go-api/server/router.go` | Register CLI MCP routes |
| `go-api/clients/auth/client.go` | Add `CliAuthorize()`, `CliResourcePermissions()` |
| `apps/api/src/api/routes/internal/index.ts` | Register cli-authorize route |
| `apps/api/src/api/routes/mcp/index.ts` | Register cli platform MCP route |
| `cli/cmd/root.go` | Register `mcp` command group |
| `cli/claude-code-plugin/.claude-plugin/plugin.json` | Add hooks + mcpServers references |
| `cli/utils/mcp.go` | Remove old app-scoped MCP config generation |

### New CLI Commands

| Command | Purpose | Visibility |
|---------|---------|------------|
| `major user token` | Print stored CLI Bearer token | Hidden |
| `major org id` | Print default org ID | Hidden |
| `major mcp check-readonly <tool>` | Check if MCP tool is read-only | Hidden |

---

### Step 8.5: Add `read_resource_context` MCP Tool

**File:** `go-common/resourcemcps/resource_context.go`

Currently `list_resource_context` only returns document metadata (name, description, size). There is no MCP tool to **read** the actual content. The ai-coder fetches docs via a separate HTTP endpoint (`GET /internal/apps/v1/:applicationId/resource-context/:resourceId/:documentId/download-url`), but this is app-scoped and not available as an MCP tool.

**Add a new tool:** `read_resource_context`

```go
{
    resourceType: "",
    name:         "read_resource_context",
    description:  "Download and return the content of a resource context document. Use after list_resource_context to read a specific document by its ID.",
    readOnly:     true,
    // Args: resourceId (string), documentId (string)
}
```

**Implementation:**
1. Validate `resourceId` and `documentId` as UUIDs
2. Call `deps.GetResourceConfig()` to verify resource access (same pattern as `list_resource_context`)
3. Call a new `deps.GetResourceContextContent()` callback that:
   - Gets a presigned S3 URL via `resourceContextService.GetPresignedURL()`
   - Fetches the content from the presigned URL
   - Returns the content (or the presigned URL for the client to fetch)
4. Return the document content as text

**New MCPToolDeps field:**
```go
GetResourceContextContent func(ctx context.Context, resourceID, documentID uuid.UUID) (string, error)
```

**Wiring in `ep_mcp.go` and `ep_mcp_cli.go`:**
```go
GetResourceContextContent: func(ctx context.Context, resourceID, documentID uuid.UUID) (string, error) {
    url, err := s.resourceContextService.GetPresignedURL(ctx, organizationID, resourceID, documentID)
    if err != nil {
        return "", err
    }
    // Fetch content from presigned URL and return it
    // Or return the URL directly for the client to fetch
    return url, nil
},
```

**No applicationId needed** — `GetPresignedURL()` only requires `orgId`, `resourceId`, `documentId`. Works in both the existing app-scoped MCP and the new org-level CLI MCP.

**New files:**
- None — added to existing `resource_context.go`

**Modified files:**

| File | Change |
|------|--------|
| `go-common/resourcemcps/resource_context.go` | Add `read_resource_context` tool |
| `go-common/resourcemcps/mcp.go` | Add `GetResourceContextContent` to `MCPToolDeps` |
| `go-api/server/ep_mcp.go` | Wire `GetResourceContextContent` callback |
| `go-api/server/ep_mcp_cli.go` | Wire `GetResourceContextContent` callback |

---

## Step 9: Migrate AI-Coder Skills — Shared Skills Directory + Two Plugin Wrappers

### Background

The ai-coder currently has ~33 skills across three directories in `mono-builder`. These are the same skills CLI users need. We want a single source of truth with two plugins that cherry-pick what they include.

**Key constraints from the plugin system:**
- Plugins can't reference files outside their root via `../` (paths don't survive cache copy)
- Symlinks ARE followed during cache copy (docs confirm this)
- `plugin.json` `skills` field accepts `string | array` — can point to specific skill subdirectories
- `mcpServers` can be omitted entirely from `plugin.json`
- A single marketplace can list multiple plugins pointing to different subdirectories

### Repo Structure

Skills live once at the top level. Each plugin directory symlinks to the shared skills and has its own `plugin.json` that selects what to include. The marketplace manifest lives at the repo root.

```
cli/
├── .claude-plugin/
│   └── marketplace.json              # Lists both plugins
├── skills/                            # Single source of truth for ALL skills
│   ├── major/                         # CLI reference skill
│   │   ├── SKILL.md
│   │   └── docs/
│   ├── resources/                     # All 24 resource connector skills
│   │   ├── postgresql/SKILL.md
│   │   ├── s3/SKILL.md
│   │   ├── slack/SKILL.md
│   │   ├── hubspot/SKILL.md
│   │   ├── salesforce/SKILL.md
│   │   ├── bigquery/SKILL.md
│   │   ├── snowflake/SKILL.md
│   │   ├── cosmosdb/SKILL.md
│   │   ├── dynamodb/SKILL.md
│   │   ├── mssql/SKILL.md
│   │   ├── neo4j/SKILL.md
│   │   ├── custom-api/SKILL.md
│   │   ├── graphql/SKILL.md
│   │   ├── lambda/SKILL.md
│   │   ├── googlesheets/SKILL.md
│   │   ├── google-analytics/SKILL.md
│   │   ├── outreach/SKILL.md
│   │   ├── quickbooks/SKILL.md
│   │   ├── dynamics/SKILL.md
│   │   ├── gong/SKILL.md
│   │   ├── ai-proxy/SKILL.md
│   │   ├── list-resources/SKILL.md
│   │   ├── majorauth/SKILL.md
│   │   └── managed-database/SKILL.md
│   ├── memory/SKILL.md
│   ├── authn/SKILL.md
│   ├── crons/SKILL.md
│   ├── webhooks/SKILL.md
│   └── agent-tools/
│       └── modifying-or-creating-agent-tools/SKILL.md
├── plugins/
│   ├── major/                         # Plugin 1: CLI users (everything)
│   │   ├── .claude-plugin/plugin.json
│   │   ├── skills → ../../skills      # symlink to shared skills
│   │   ├── .mcp.json                  # headersHelper for org-level MCPs
│   │   ├── hooks/hooks.json
│   │   └── scripts/
│   │       ├── get-headers.sh
│   │       └── check-readonly.sh
│   └── major-platform/               # Plugin 2: AI-coder (skills only)
│       ├── .claude-plugin/plugin.json
│       └── skills → ../../skills      # symlink to shared skills
```

### Plugin Manifests

**`plugins/major/.claude-plugin/plugin.json`** — CLI users get everything:
```json
{
  "name": "major",
  "description": "Use the Major platform agentically via Claude Code",
  "version": "1.0.0",
  "author": { "name": "Major Technology" },
  "skills": "./skills/",
  "mcpServers": "./.mcp.json",
  "hooks": "./hooks/hooks.json"
}
```

**`plugins/major-platform/.claude-plugin/plugin.json`** — AI-coder gets only platform skills (no MCP, no hooks, no CLI skill):
```json
{
  "name": "major-platform",
  "description": "Major platform resource connectors and development skills",
  "version": "1.0.0",
  "author": { "name": "Major Technology" },
  "skills": [
    "./skills/resources/",
    "./skills/memory/",
    "./skills/authn/",
    "./skills/crons/",
    "./skills/webhooks/",
    "./skills/agent-tools/"
  ]
}
```

Note: `major-platform` omits `mcpServers` and `hooks` entirely — the ai-coder has its own MCP setup. It also excludes `./skills/major/` (the CLI reference skill) since the ai-coder doesn't have the Major CLI installed.

### Marketplace Manifest

**`cli/.claude-plugin/marketplace.json`** — at the repo root, lists both plugins:
```json
{
  "name": "major-tools",
  "owner": {
    "name": "Major Technology",
    "email": "support@major.build"
  },
  "metadata": {
    "description": "Official Major platform plugins for Claude Code",
    "version": "1.0.0"
  },
  "plugins": [
    {
      "name": "major",
      "source": "./plugins/major",
      "description": "Use the Major platform agentically via Claude Code"
    },
    {
      "name": "major-platform",
      "source": "./plugins/major-platform",
      "description": "Major platform resource connectors and development skills"
    }
  ]
}
```

### How Symlinks Work at Install Time

When a CLI user runs `/plugin install major@major-tools`:
1. Claude Code copies the `plugins/major/` directory to the plugin cache
2. The `skills` symlink is followed — the shared skills directory contents are copied into the cached plugin
3. The plugin works with its own copy of all skills

Same for the ai-coder loading `major-platform` locally — symlinks resolve at copy time.

### AI-Coder Integration

**Dockerfile change:** Copy the plugin into the Docker image at build time.

```dockerfile
COPY --from=plugin-builder /cli/plugins/major-platform /app/plugins/major-platform
```

**Session options change** in `apps/ai-coder/src/base/startup.ts`:

```typescript
const plugins: SdkPluginConfig[] = [
  { type: 'local', path: '/app/plugins/major-platform' }
];

return {
  ...sessionOptions,
  plugins,
};
```

**Remove from ai-coder:**
- Delete `apps/ai-coder/skills_shared/resources/` (moved to plugin)
- Delete `apps/ai-coder/skills/resources/` (moved to plugin)
- Delete `apps/ai-coder/src/agents/coder/skills/authn/`, `crons/`, `webhooks/` (moved to plugin)
- Delete `apps/ai-coder/src/agents/general-agent/skills/modifying-or-creating-agent-tools/` (moved to plugin)
- Simplify `copySkillsFlattened()` in `startup.ts` — only needed for ai-coder-specific skills

**Skills that stay in ai-coder** (not moved — ai-coder-specific):
- `skills_shared/new-project/` — project scaffolding flow
- `skills_shared/session-files/` — coding session file operations
- `src/agents/coder/skills/frontend/data-table/` — frontend UI generation
- Dynamic theme skill — fetched from API at runtime

### What Each User Gets

| User | Installs | Gets |
|------|----------|------|
| CLI user | `major` plugin | CLI skill + all resource skills + MCP servers + hooks |
| AI-coder | `major-platform` plugin (loaded locally) | Resource + platform skills only |

CLI users install one plugin and get everything. No need to install two plugins.

### Migration Steps

1. Create `cli/skills/` directory, copy all skills from ai-coder
2. Update any skill content that references ai-coder-specific paths
3. Create `plugins/major/` and `plugins/major-platform/` with symlinks + manifests
4. Move existing `claude-code-plugin/` content into `plugins/major/`
5. Create `cli/.claude-plugin/marketplace.json` at repo root
6. Update ai-coder to load plugin via `plugins: [{ type: 'local', path: '...' }]`
7. Remove migrated skills from ai-coder
8. Update ai-coder Dockerfile to include the plugin
9. Test both flows (CLI install + ai-coder session)

---

### New Files (Step 9)

| File | Repo | Purpose |
|------|------|---------|
| `cli/skills/**` | cli | Shared skills directory (single source of truth) |
| `cli/plugins/major/` | cli | CLI plugin wrapper (symlink + MCP + hooks) |
| `cli/plugins/major-platform/` | cli | AI-coder plugin wrapper (symlink, skills only) |
| `cli/.claude-plugin/marketplace.json` | cli | Marketplace manifest listing both plugins |

### Modified Files (Step 9)

| File | Change |
|------|--------|
| `apps/ai-coder/src/base/startup.ts` | Load plugin instead of copying skills |
| `apps/ai-coder/Dockerfile` | Add plugin to image |
| `apps/ai-coder/skills_shared/resources/` | Delete (moved to cli/skills/) |
| `apps/ai-coder/skills/resources/` | Delete (moved to cli/skills/) |
| `apps/ai-coder/src/agents/coder/skills/` | Delete authn, crons, webhooks (moved) |
| `cli/claude-code-plugin/` | Moved to `cli/plugins/major/` |

---

## Remaining Research Items

1. **CLI token expiry** — How long do CLI tokens last? If short-lived, `headersHelper` handles refresh automatically (runs on each connection). If very short, may need a refresh mechanism.
2. **CORS for CLI MCP routes** — Claude Code makes HTTP requests to MCP servers. Verify go-api/Node API accept requests without Origin headers (should be fine since these aren't browser requests).

## Out of Scope (v1)

- Token auto-refresh (headersHelper runs on each connection, which should be sufficient)
- Per-app environment selection via CLI MCP (uses org default environment)
- Filtering tools by connected resources
- Chat history in CLI MCP (can be added later as a tool on the platform MCP)
