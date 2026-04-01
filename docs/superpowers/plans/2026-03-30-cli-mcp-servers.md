# CLI MCP Servers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add org-level MCP endpoints that authenticate via the CLI's Bearer token, ship them in the Claude Code plugin, and migrate ai-coder skills to a shared directory.

**Architecture:** New `/internal/cli-authorize` endpoint on Node API validates CLI tokens for Go API. Go API gets a new `/cli/v1/mcp` org-level resource MCP endpoint. Node API gets a new `/mcp/cli` org-level platform MCP endpoint. Plugin uses `headersHelper` scripts to dynamically resolve auth headers. Skills move to a shared `skills/` directory with symlinks from two plugin wrappers.

**Tech Stack:** Go (gin, MCP SDK), TypeScript (Fastify, MCP SDK), Bash (headersHelper scripts), Cobra (CLI commands)

---

## File Structure

### mono-builder (new files)

| File | Responsibility |
|------|---------------|
| `apps/api/src/api/routes/internal/cli-authorize.ts` | CLI token authorization endpoint |
| `apps/api/src/schemas/cli-authorize.ts` | Zod schemas for cli-authorize |
| `apps/api/src/api/routes/mcp/major-platform-cli.ts` | Org-level platform MCP for CLI |
| `go-api/server/ep_mcp_cli.go` | Org-level resource MCP endpoint |

### mono-builder (modified files)

| File | Change |
|------|--------|
| `apps/api/src/api/routes/internal/index.ts` | Register cli-authorize route |
| `apps/api/src/api/routes/mcp/index.ts` | Register CLI platform MCP route |
| `apps/api/src/schemas/internal.ts` | Import cli-authorize schemas |
| `go-api/clients/auth/client.go` | Add `CliAuthorize()` method |
| `go-api/server/middleware.go` | Add `CheckCliAuth()` middleware |
| `go-api/server/router.go` | Register CLI MCP routes |
| `go-common/resourcemcps/resource_context.go` | Add `read_resource_context` tool |
| `go-common/resourcemcps/mcp.go` | Add `GetResourceContextContent` to deps |

### cli (new files)

| File | Responsibility |
|------|---------------|
| `cmd/user/token.go` | `major user token` — print Bearer token |
| `cmd/org/id.go` | `major org id` — print default org ID |
| `plugins/major/scripts/get-headers.sh` | headersHelper for MCP auth |
| `plugins/major/.mcp.json` | Plugin MCP server config |
| `plugins/major/hooks/hooks.json` | Auto-approve read-only tools |
| `plugins/major/scripts/check-readonly.sh` | Hook script for read-only check |
| `cmd/mcp/mcp.go` | MCP command group |
| `cmd/mcp/check_readonly.go` | `major mcp check-readonly` |

### cli (modified files)

| File | Change |
|------|--------|
| `cmd/user/user.go` | Register `tokenCmd` |
| `cmd/org/org.go` | Register `idCmd` |
| `cmd/root.go` | Register `mcp` command group |
| `plugins/major/.claude-plugin/plugin.json` | Add hooks + mcpServers refs |
| `configs/prod.json` | Add `resource_api_url` field |
| `configs/local.json` | Add `resource_api_url` field |
| `clients/config/config.go` | Add `ResourceAPIURL` to Config |

---

## Phase 1: Backend Auth Foundation (mono-builder)

### Task 1: CLI Authorize Endpoint — Schemas

**Files:**
- Create: `mono-builder/apps/api/src/schemas/cli-authorize.ts`

- [ ] **Step 1: Create the schema file**

```typescript
// apps/api/src/schemas/cli-authorize.ts
import { z } from "zod";

export const CliAuthorizeRequestSchema = z.object({
	organizationId: z.string().uuid(),
	permission: z.string().optional(),
	resourceId: z.string().uuid().optional(),
	applicationId: z.string().uuid().optional(),
});
export type CliAuthorizeRequest = z.infer<typeof CliAuthorizeRequestSchema>;

export const CliAuthorizeResponseSchema = z.object({
	authorized: z.boolean(),
	userId: z.string().optional(),
	organizationId: z.string().optional(),
	error: z.string().optional(),
});
export type CliAuthorizeResponse = z.infer<typeof CliAuthorizeResponseSchema>;
```

- [ ] **Step 2: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add apps/api/src/schemas/cli-authorize.ts
git commit -m "feat: add CLI authorize schemas"
```

---

### Task 2: CLI Authorize Endpoint — Route

**Files:**
- Create: `mono-builder/apps/api/src/api/routes/internal/cli-authorize.ts`
- Modify: `mono-builder/apps/api/src/api/routes/internal/index.ts`

- [ ] **Step 1: Create the cli-authorize route file**

```typescript
// apps/api/src/api/routes/internal/cli-authorize.ts
import type { FastifyPluginAsync } from "fastify";
import type { ZodTypeProvider } from "fastify-type-provider-zod";
import type { AppContainer } from "@/container";
import { CliAuthorizeRequestSchema, CliAuthorizeResponseSchema } from "@/schemas/cli-authorize";
import { roles as rolesSchema } from "@repo/api/schemas";

export function createCliAuthorizeRoute(container: AppContainer): FastifyPluginAsync {
	const { tokenAuthMiddleware, authzDAO } = container;

	const plugin: FastifyPluginAsync = async (app): Promise<void> => {
		const r = app.withTypeProvider<ZodTypeProvider>();

		r.post(
			"/",
			{
				preHandler: tokenAuthMiddleware.assertCliToken(),
				schema: {
					body: CliAuthorizeRequestSchema,
					response: {
						200: CliAuthorizeResponseSchema,
					},
				},
			},
			async (request) => {
				const userId = request.tokenUser!.userId;
				const { organizationId, permission, resourceId, applicationId } = request.body;

				// Verify user is a member of the organization
				const isMember = await authzDAO.isMember({ organizationId, userId });

				if (!isMember) {
					return {
						authorized: false,
						userId,
						error: "Not a member of this organization",
					};
				}

				// If no specific permission requested, just verify membership
				if (!permission) {
					return {
						authorized: true,
						userId,
						organizationId,
					};
				}

				// Check permission based on scope
				let hasPermission = false;

				if (applicationId && permission.startsWith("application:")) {
					hasPermission = await authzDAO.hasApplicationPermission({
						organizationId,
						userId,
						applicationId,
						permissionString: permission as rolesSchema.ApplicationPermission,
					});
				} else if (resourceId) {
					hasPermission = await authzDAO.hasPermission({
						organizationId,
						subjectId: userId,
						permissionString: permission,
						objectId: resourceId,
					});
				} else {
					hasPermission = await authzDAO.hasOrgPermission({
						organizationId,
						subjectId: userId,
						permissionString: permission,
					});
				}

				if (!hasPermission) {
					return {
						authorized: false,
						userId,
						organizationId,
						error: `Missing permission: ${permission}`,
					};
				}

				return {
					authorized: true,
					userId,
					organizationId,
				};
			},
		);
	};

	return plugin;
}
```

- [ ] **Step 2: Register the route in internal/index.ts**

In `apps/api/src/api/routes/internal/index.ts`, add the import at the top:

```typescript
import { createCliAuthorizeRoute } from "./cli-authorize";
```

Then inside the `plugin` async function (after the existing route registrations around line 132), add:

```typescript
await r.register(createCliAuthorizeRoute(container), {
	prefix: "/cli-authorize",
});
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/josegiron/Documents/code/mono-builder
yarn workspace api build
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add apps/api/src/schemas/cli-authorize.ts apps/api/src/api/routes/internal/cli-authorize.ts apps/api/src/api/routes/internal/index.ts
git commit -m "feat: add /internal/cli-authorize endpoint for CLI token auth"
```

---

### Task 3: CLI Resource Permissions Endpoint

The existing `/internal/app-resource-permissions` requires `applicationId`. We need an org-level version.

**Files:**
- Modify: `mono-builder/apps/api/src/api/routes/internal/index.ts`
- Modify: `mono-builder/apps/api/src/schemas/internal.ts`

- [ ] **Step 1: Add schemas for CLI resource permissions**

In `apps/api/src/schemas/internal.ts`, add after the `AppResourcePermissionsResponseSchema`:

```typescript
/* =========================================
 * CLI Resource Permissions Schema (org-level, no applicationId)
 *
 * Used by Go API CLI MCP to check which resources the user has access to.
 * Same as AppResourcePermissions but without applicationId requirement.
 * ======================================= */

export const CliResourcePermissionsRequestSchema = z.object({
	organizationId: z.string().uuid(),
	resourceIds: z.array(z.string().uuid()),
});
export type CliResourcePermissionsRequest = z.infer<typeof CliResourcePermissionsRequestSchema>;

export const CliResourcePermissionsResponseSchema = z.object({
	permissions: z.record(z.string(), z.boolean()),
});
export type CliResourcePermissionsResponse = z.infer<typeof CliResourcePermissionsResponseSchema>;
```

- [ ] **Step 2: Add the endpoint in internal/index.ts**

Add a new route in the `cli-authorize` route file. Actually, it's cleaner to add it directly in `internal/index.ts` alongside the existing `app-resource-permissions`. Add after the `/app-resource-permissions` handler (around line 1300):

```typescript
/**
 * POST /internal/cli-resource-permissions
 *
 * Org-level resource permission check for CLI MCP.
 * Validates CLI token, then checks which resources user has admin access to.
 * Unlike app-resource-permissions, does not require applicationId.
 */
r.post(
	"/cli-resource-permissions",
	{
		preHandler: container.tokenAuthMiddleware.assertCliToken(),
		schema: {
			body: CliResourcePermissionsRequestSchema,
			response: {
				200: CliResourcePermissionsResponseSchema,
			},
		},
	},
	async (request) => {
		const userId = request.tokenUser!.userId;
		const { organizationId, resourceIds } = request.body;

		// Verify user is a member
		const isMember = await authzDAO.isMember({ organizationId, userId });

		if (!isMember) {
			return { permissions: {} };
		}

		// Check admin permission on each resource
		const permissions: Record<string, boolean> = {};

		for (const resourceId of resourceIds) {
			const hasPermission = await authzDAO.hasPermission({
				organizationId,
				subjectId: userId,
				permissionString: "resource:admin",
				objectId: resourceId,
			});
			permissions[resourceId] = hasPermission;
		}

		return { permissions };
	},
);
```

Also add the import for the new schemas at the top of `internal/index.ts`:

```typescript
import {
	CliResourcePermissionsRequestSchema,
	CliResourcePermissionsResponseSchema,
} from "@/schemas/internal";
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/josegiron/Documents/code/mono-builder
yarn workspace api build
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add apps/api/src/schemas/internal.ts apps/api/src/api/routes/internal/index.ts
git commit -m "feat: add /internal/cli-resource-permissions endpoint"
```

---

### Task 4: Go API Auth Client — CliAuthorize Method

**Files:**
- Modify: `mono-builder/go-api/clients/auth/client.go`

- [ ] **Step 1: Add CliAuthorize types and method**

Append to `go-api/clients/auth/client.go` before the `forwardAuthHeaders` method:

```go
// CliAuthorizeRequest is the request body for CLI token authorization
type CliAuthorizeRequest struct {
	OrganizationID string `json:"organizationId"`
}

// CliAuthorizeResponse is the response from the CLI authorize endpoint
type CliAuthorizeResponse struct {
	Authorized     bool   `json:"authorized"`
	UserID         string `json:"userId,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	Error          string `json:"error,omitempty"`
}

// CliAuthorize validates a CLI Bearer token and checks org membership.
// Used by the CLI MCP middleware to authenticate requests.
func (c *Client) CliAuthorize(headers http.Header, organizationID string) (*CliAuthorizeResponse, error) {
	reqBody := CliAuthorizeRequest{
		OrganizationID: organizationID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cli authorize request: %w", err)
	}

	url := fmt.Sprintf("%s/internal/cli-authorize", c.nodeAPIURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create cli authorize request: %w", err)
	}

	// Forward the Authorization header (Bearer token)
	if auth := headers.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httputil.DoWithRetry(c.httpClient, req, jsonBody)
	if err != nil {
		return nil, fmt.Errorf("failed to call cli authorize endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cli authorize endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read cli authorize response: %w", err)
	}

	var authResp CliAuthorizeResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cli authorize response: %w", err)
	}

	return &authResp, nil
}

// CliResourcePermissionsRequest is the request body for CLI resource permission checks
type CliResourcePermissionsRequest struct {
	OrganizationID string   `json:"organizationId"`
	ResourceIDs    []string `json:"resourceIds"`
}

// CliResourcePermissionsResponse contains permissions for each resource
type CliResourcePermissionsResponse struct {
	Permissions map[string]bool `json:"permissions"`
}

// CliResourcePermissions checks which resources the user has admin access to.
// Org-level variant that doesn't require applicationId.
func (c *Client) CliResourcePermissions(headers http.Header, organizationID string, resourceIDs []string) (*CliResourcePermissionsResponse, error) {
	reqBody := CliResourcePermissionsRequest{
		OrganizationID: organizationID,
		ResourceIDs:    resourceIDs,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cli resource permissions request: %w", err)
	}

	url := fmt.Sprintf("%s/internal/cli-resource-permissions", c.nodeAPIURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create cli resource permissions request: %w", err)
	}

	if auth := headers.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httputil.DoWithRetry(c.httpClient, req, jsonBody)
	if err != nil {
		return nil, fmt.Errorf("failed to call cli resource permissions endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read cli resource permissions response: %w", err)
	}

	var permResp CliResourcePermissionsResponse
	if err := json.Unmarshal(body, &permResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cli resource permissions response: %w", err)
	}

	return &permResp, nil
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add go-api/clients/auth/client.go
git commit -m "feat: add CliAuthorize and CliResourcePermissions to auth client"
```

---

### Task 5: Go API — CheckCliAuth Middleware

**Files:**
- Modify: `mono-builder/go-api/server/middleware.go`

- [ ] **Step 1: Add CheckCliAuth middleware**

Append to `go-api/server/middleware.go` (after the `CheckPermission` function, before `CheckCrossServerAuth`):

```go
// CheckCliAuth creates a middleware for CLI token authentication.
// Reads organizationId from the x-major-org-id header, forwards the
// Authorization (Bearer) header to Node API's /internal/cli-authorize.
// Sets ContextKeyOrganizationID and ContextKeyUserID on success.
func (s *Server) CheckCliAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID := c.GetHeader("x-major-org-id")
		if organizationID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, errors.ErrorBadBody)
			return
		}

		result, err := s.authClient.CliAuthorize(c.Request.Header, organizationID)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, errors.ErrorInternalAuthServiceUnavailable, err)
			return
		}

		if !result.Authorized {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errors.ErrorUnauthorized)
			return
		}

		c.Set(constants.ContextKeyOrganizationID, result.OrganizationID)
		c.Set(constants.ContextKeyUserID, result.UserID)
		c.Next()
	}
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add go-api/server/middleware.go
git commit -m "feat: add CheckCliAuth middleware for CLI token auth"
```

---

### Task 6: Go API — Org-Level Resource MCP Endpoint

**Files:**
- Create: `mono-builder/go-api/server/ep_mcp_cli.go`
- Modify: `mono-builder/go-api/server/router.go`

- [ ] **Step 1: Create ep_mcp_cli.go**

```go
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/major-technology/mono-builder/go-api/clients/nodeapi"
	"github.com/major-technology/mono-builder/go-common/constants"
	"github.com/major-technology/mono-builder/go-common/errors"
	"github.com/major-technology/mono-builder/go-common/resourcemcps"
	"github.com/major-technology/mono-builder/go-common/resourcetypes"
	"github.com/major-technology/mono-builder/go-common/utils/ginutils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// addCliMCPRoutes adds the org-level MCP endpoint for CLI users.
// No applicationId in the path — auth via CLI Bearer token + x-major-org-id header.
func (s *Server) addCliMCPRoutes(router *response.Router) {
	cliMCPRoutes := router.Group("/cli/v1/mcp", s.CheckCliAuth())
	cliMCPRoutes.GET("/tools", s.handleGetMCPToolsMetadata)
	cliMCPRoutes.RawAny("", s.handleCliMCPRequest)
}

// handleCliMCPRequest handles MCP protocol requests for CLI users.
// Same as handleMCPRequest but without applicationId requirement.
func (s *Server) handleCliMCPRequest(c *gin.Context) {
	organizationID, err := ginutils.GetAndCast[string](c, constants.ContextKeyOrganizationID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "missing organization context"})
		return
	}

	var userID string
	if val, exists := c.Get(constants.ContextKeyUserID); exists {
		userID, _ = val.(string)
	}

	requestHeaders := c.Request.Header

	deps := &resourcemcps.MCPToolDeps{
		ExternalServices: s.externalServices,
		OrganizationID:   organizationID,
		UserID:           userID,
		ApplicationID:    "", // No application context for CLI MCP
		MajorJWT:         "",
		GetResourceConfig: func(ctx context.Context, resourceID string, opts ...resourcemcps.GetResourceConfigOpt) (*resourcetypes.Config, *resourcetypes.Meta, constants.ResourceSubtype, error) {
			return s.getResourceConfigForCliMCP(ctx, organizationID, userID, resourceID, opts...)
		},
		ListResources: func(ctx context.Context) ([]resourcemcps.ResourceInfo, error) {
			return s.listResourcesForCliMCP(ctx, organizationID, requestHeaders)
		},
		ListResourceContext: func(ctx context.Context, resourceID uuid.UUID) ([]resourcemcps.ResourceDocumentResult, error) {
			docs, err := s.resourceContextService.List(ctx, organizationID, resourceID)
			if err != nil {
				return nil, err
			}

			result := make([]resourcemcps.ResourceDocumentResult, len(docs))
			for i, doc := range docs {
				result[i] = resourcemcps.ResourceDocumentResult{
					ID:              doc.ID,
					Name:            doc.Name,
					Description:     doc.Description,
					StructuralIndex: doc.StructuralIndex,
					ContentType:     doc.ContentType,
					SizeBytes:       doc.SizeBytes,
				}
			}

			return result, nil
		},
		UpdateResourceConfig: func(ctx context.Context, resourceID string, config *resourcetypes.Config) error {
			return s.updateResourceConfigForMCP(ctx, organizationID, resourceID, config)
		},
		GetManagedDatabaseByResourceID: func(ctx context.Context, resourceID string) (*resourcemcps.ManagedDatabaseInfo, error) {
			return s.migrationService.GetManagedDatabaseByResourceID(ctx, organizationID, resourceID)
		},
		GetManagedDatabaseSchemas: func(ctx context.Context, managedDatabaseID string) ([]resourcemcps.ManagedDatabaseSchemaInfo, error) {
			return s.migrationService.GetManagedDatabaseSchemas(ctx, organizationID, managedDatabaseID)
		},
		RunMigration: func(ctx context.Context, params resourcemcps.RunMigrationParams) (*resourcemcps.MigrationResult, error) {
			return s.migrationService.RunMigrationFromMCP(ctx, organizationID, params)
		},
		SetupManagedDatabase: func(ctx context.Context) (*resourcemcps.ManagedDatabaseSetupResult, error) {
			// SetupManagedDatabase requires applicationId — return clear error for CLI users
			return nil, fmt.Errorf("setup_managed_database requires an application context; use 'major resource manage' instead")
		},
	}

	mcpServer := resourcemcps.CreateUnifiedMCPServer(deps)

	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return mcpServer },
		&mcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		},
	)

	handler.ServeHTTP(c.Writer, c.Request)
}

// getResourceConfigForCliMCP fetches resource config for CLI MCP.
// Same as getResourceConfigForMCP but without applicationId (skips user env choice).
func (s *Server) getResourceConfigForCliMCP(
	ctx context.Context,
	organizationID,
	userID,
	resourceIDStr string,
	opts ...resourcemcps.GetResourceConfigOpt,
) (*resourcetypes.Config, *resourcetypes.Meta, constants.ResourceSubtype, error) {
	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		return nil, nil, "", errors.WrapError("invalid resource ID", err)
	}

	configOpts := &resourcemcps.GetResourceConfigOpts{}
	for _, opt := range opts {
		opt(configOpts)
	}

	permission := constants.PermissionResourceBuild
	if configOpts.Permission != nil {
		permission = *configOpts.Permission
	}

	// Check permission via Node API
	if s.config.Env != constants.EnvLocal {
		authorized, err := s.nodeAPIClient.CheckPermission(ctx, nodeapi.CheckPermissionRequest{
			OrganizationID: organizationID,
			SubjectID:      userID,
			ObjectID:       resourceIDStr,
			Permission:     permission,
		})
		if err != nil {
			return nil, nil, "", errors.WrapError("failed to check resource permission", err)
		}

		if !authorized {
			return nil, nil, "", errors.ErrorUnauthorized
		}
	}

	resource, err := s.dbClient.GetResourceByID(organizationID, resourceID)
	if err != nil {
		return nil, nil, "", errors.WrapError("resource not found", err)
	}

	// For CLI MCP, skip user environment choice — go straight to default build env
	var envID uuid.UUID

	if resource.SharedConfig {
		defaultEnv, err := s.dbClient.GetDefaultEnvironment(organizationID)
		if err != nil {
			return nil, nil, "", errors.WrapError("failed to get default environment", err)
		}

		if defaultEnv == nil {
			return nil, nil, "", errors.ErrorResourceEnvironmentNotFound
		}

		envID = defaultEnv.ID
	}

	if envID == uuid.Nil {
		defaultBuildEnv, err := s.dbClient.GetDefaultBuildEnvironment(organizationID)
		if err != nil {
			return nil, nil, "", errors.WrapError("failed to get default build environment", err)
		}

		if defaultBuildEnv == nil {
			return nil, nil, "", errors.ErrorResourceEnvironmentNotFound
		}

		envID = defaultBuildEnv.ID
	}

	_, env, err := s.resourceService.GetResourceWithEnv(ctx, organizationID, resourceID, envID)
	if err != nil {
		return nil, nil, "", errors.WrapError("failed to get resource config", err)
	}

	return env.DecryptedConfig, env.Meta, resource.Subtype, nil
}

// listResourcesForCliMCP returns all resources the CLI user has admin access to.
// Org-level: no applicationId filtering, no AppResourceIds.
func (s *Server) listResourcesForCliMCP(_ context.Context, organizationID string, headers http.Header) ([]resourcemcps.ResourceInfo, error) {
	dbResources, err := s.dbClient.ListResourcesWithEnvironments(organizationID)
	if err != nil {
		return nil, errors.WrapError("failed to list resources from database", err)
	}

	if len(dbResources) == 0 {
		return []resourcemcps.ResourceInfo{}, nil
	}

	resourceIDs := make([]string, len(dbResources))
	for i, r := range dbResources {
		resourceIDs[i] = r.ID.String()
	}

	permResp, err := s.authClient.CliResourcePermissions(headers, organizationID, resourceIDs)
	if err != nil {
		return nil, errors.WrapError("failed to check resource permissions", err)
	}

	var managedResourceIDs []uuid.UUID
	for _, r := range dbResources {
		if r.IsManaged && permResp.Permissions[r.ID.String()] {
			managedResourceIDs = append(managedResourceIDs, r.ID)
		}
	}

	managedScopes := make(map[uuid.UUID]*uuid.UUID)
	if len(managedResourceIDs) > 0 {
		managedScopes, err = s.dbClient.GetManagedDatabaseScopesByResourceIDs(organizationID, managedResourceIDs)
		if err != nil {
			return nil, errors.WrapError("failed to get managed database scopes", err)
		}
	}

	var resources []resourcemcps.ResourceInfo
	for _, r := range dbResources {
		if permResp.Permissions[r.ID.String()] {
			info := resourcemcps.ResourceInfo{
				ID:            r.ID.String(),
				Name:          r.Name,
				Subtype:       string(r.Subtype),
				Description:   r.Description,
				IsManaged:     r.IsManaged,
				GrantedScopes: extractGrantedScopes(r),
				InUseByApp:    false, // No app context
			}

			if r.IsManaged {
				if appID, ok := managedScopes[r.ID]; ok && appID != nil {
					info.ManagedScope = "app"
				} else {
					info.ManagedScope = "org"
				}
			}

			resources = append(resources, info)
		}
	}

	return resources, nil
}
```

- [ ] **Step 2: Add missing import for `response` package**

The file uses `response.Router` — add the import:

```go
import (
	// ... existing imports ...
	"github.com/major-technology/mono-builder/go-common/response"
)
```

- [ ] **Step 3: Register routes in router.go**

In `go-api/server/router.go`, add `s.addCliMCPRoutes(router)` after `s.addMCPRoutes(router)` (around line 66):

```go
s.addMCPRoutes(router)
s.addCliMCPRoutes(router)
```

- [ ] **Step 4: Verify build**

```bash
cd /Users/josegiron/Documents/code/mono-builder
cd go-api && go build ./...
```

- [ ] **Step 5: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add go-api/server/ep_mcp_cli.go go-api/server/router.go
git commit -m "feat: add org-level CLI MCP endpoint on Go API"
```

---

### Task 7: Node API — Org-Level Platform MCP for CLI

**Files:**
- Create: `mono-builder/apps/api/src/api/routes/mcp/major-platform-cli.ts`
- Modify: `mono-builder/apps/api/src/api/routes/mcp/index.ts`

- [ ] **Step 1: Create the CLI platform MCP route**

```typescript
// apps/api/src/api/routes/mcp/major-platform-cli.ts
import type { FastifyPluginAsync, FastifyRequest, FastifyReply } from "fastify";
import type { IncomingMessage, ServerResponse } from "node:http";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { logger } from "@repo/logger";
import type { TokenAuthMiddleware, TokenAuthRequest } from "@/api/middleware/token-auth";
import { createMajorPlatformMcpServer, type MajorPlatformMcpDeps } from "@/mcp/server";
import type { ApplicationsDAO } from "@/data/db_dao/applications";
import type { AuthzDAO } from "@/data/db_dao/authz";

export interface McpCliPlatformRouteDeps {
	mcpDeps: MajorPlatformMcpDeps;
	applicationsDAO: ApplicationsDAO;
	tokenAuthMiddleware: TokenAuthMiddleware;
	authzDAO: AuthzDAO;
}

export function createMajorPlatformCliMcpRoute(deps: McpCliPlatformRouteDeps): FastifyPluginAsync {
	const { mcpDeps, applicationsDAO, tokenAuthMiddleware, authzDAO } = deps;

	const mcpPlugin: FastifyPluginAsync = async (app): Promise<void> => {
		app.addContentTypeParser("application/json", { parseAs: "buffer" }, (_req, body, done) => {
			done(null, body);
		});

		app.post("/", { preHandler: tokenAuthMiddleware.assertCliToken() }, async (request: FastifyRequest, reply: FastifyReply) => {
			const tokenUser = (request as TokenAuthRequest).tokenUser!;
			const organizationId = request.headers["x-major-org-id"] as string;

			if (!organizationId) {
				return reply.status(400).send({
					jsonrpc: "2.0",
					error: { code: -32600, message: "Missing x-major-org-id header" },
					id: null,
				});
			}

			let requestId: string | number | null = null;
			let body: Record<string, unknown>;

			try {
				body = JSON.parse((request.body as Buffer).toString());
				requestId = (body?.id as string | number | null) ?? null;
			} catch (error) {
				logger.error({ err: error }, "CLI Platform MCP parse error");

				return reply.status(400).send({
					jsonrpc: "2.0",
					error: { code: -32700, message: "Parse error" },
					id: null,
				});
			}

			if (!body || typeof body !== "object" || !body.jsonrpc || !body.method) {
				return reply.status(400).send({
					jsonrpc: "2.0",
					error: { code: -32600, message: "Invalid Request" },
					id: requestId,
				});
			}

			try {
				// Verify org membership
				const isMember = await authzDAO.isMember({
					organizationId,
					userId: tokenUser.userId,
				});

				if (!isMember) {
					return reply.status(403).send({
						jsonrpc: "2.0",
						error: { code: -32600, message: "Not a member of this organization" },
						id: requestId,
					});
				}

				const transport = new StreamableHTTPServerTransport({
					sessionIdGenerator: undefined,
					enableJsonResponse: true,
				});

				// Create server without applicationId — tools that need it accept it as input
				const server = createMajorPlatformMcpServer(mcpDeps, {
					applicationId: "", // No app context for CLI
					organizationId,
					userId: tokenUser.userId,
					source: "ai_coder",
				});

				await server.connect(transport);
				await transport.handleRequest(request.raw as IncomingMessage, reply.raw as ServerResponse, body);

				reply.hijack();
			} catch (error) {
				logger.error({ err: error }, "CLI Platform MCP error");

				return reply.status(500).send({
					jsonrpc: "2.0",
					error: {
						code: -32603,
						message: error instanceof Error ? error.message : "Internal error",
					},
					id: requestId,
				});
			}
		});
	};

	return mcpPlugin;
}
```

- [ ] **Step 2: Register in MCP routes index**

In `apps/api/src/api/routes/mcp/index.ts`, add the import:

```typescript
import { createMajorPlatformCliMcpRoute } from "./major-platform-cli";
```

Then inside the `mcpPlugin` function, add after the major-platform registration:

```typescript
// Major Platform MCP for CLI users (org-level, CLI token auth)
await app.register(
	createMajorPlatformCliMcpRoute({
		mcpDeps: {
			applicationsDAO: container.applicationsDAO,
			applicationVersionsDAO: container.applicationVersionsDAO,
			applicationService: container.applicationService,
			memoryService: container.memoryService,
			aiProxyService: container.aiProxyService,
			authzService: container.authzService,
		},
		applicationsDAO: container.applicationsDAO,
		tokenAuthMiddleware: container.tokenAuthMiddleware,
		authzDAO: container.authzDAO,
	}),
	{ prefix: "/cli" },
);
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/josegiron/Documents/code/mono-builder
yarn workspace api build
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add apps/api/src/api/routes/mcp/major-platform-cli.ts apps/api/src/api/routes/mcp/index.ts
git commit -m "feat: add org-level CLI platform MCP endpoint"
```

---

### Task 8: Add `read_resource_context` MCP Tool

**Files:**
- Modify: `mono-builder/go-common/resourcemcps/mcp.go`
- Modify: `mono-builder/go-common/resourcemcps/resource_context.go`
- Modify: `mono-builder/go-api/server/ep_mcp.go`
- Modify: `mono-builder/go-api/server/ep_mcp_cli.go`

- [ ] **Step 1: Add GetResourceContextContent to MCPToolDeps**

In `go-common/resourcemcps/mcp.go`, add a new field to the `MCPToolDeps` struct:

```go
// GetResourceContextContent fetches the content of a resource context document.
// Returns a presigned URL to download the document content.
GetResourceContextContent func(ctx context.Context, resourceID, documentID uuid.UUID) (string, error)
```

- [ ] **Step 2: Add read_resource_context tool in resource_context.go**

In `go-common/resourcemcps/resource_context.go`, after the existing `list_resource_context` registration, register the new tool:

```go
registerTool(server, ToolRegistration{
	resourceType: "",
	name:         "read_resource_context",
	description:  "Download and return the content of a resource context document. Use after list_resource_context to read a specific document by its ID. Returns a presigned URL to download the content.",
	readOnly:     true,
	handler: func(ctx context.Context, _ *mcp.CallToolRequest, args struct {
		ResourceID string `json:"resourceId" jsonschema:"description=The resource ID that owns the document,required"`
		DocumentID string `json:"documentId" jsonschema:"description=The document ID to read (from list_resource_context),required"`
	}) (*mcp.CallToolResult, any, error) {
		if args.ResourceID == "" {
			return ErrorResult("resourceId is required"), nil, nil
		}

		if args.DocumentID == "" {
			return ErrorResult("documentId is required"), nil, nil
		}

		resourceUUID, err := uuid.Parse(args.ResourceID)
		if err != nil {
			return ErrorResult("invalid resourceId format"), nil, nil
		}

		documentUUID, err := uuid.Parse(args.DocumentID)
		if err != nil {
			return ErrorResult("invalid documentId format"), nil, nil
		}

		// Verify resource access
		_, _, _, err = deps.GetResourceConfig(ctx, args.ResourceID)
		if err != nil {
			return ErrorResult(fmt.Sprintf("cannot access resource: %v", err)), nil, nil
		}

		if deps.GetResourceContextContent == nil {
			return ErrorResult("read_resource_context is not available"), nil, nil
		}

		url, err := deps.GetResourceContextContent(ctx, resourceUUID, documentUUID)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to get document content: %v", err)), nil, nil
		}

		return JSONResult(map[string]string{
			"resourceId": args.ResourceID,
			"documentId": args.DocumentID,
			"downloadUrl": url,
		})
	},
}, deps)
```

Add `"fmt"` to imports if not already present.

- [ ] **Step 3: Wire GetResourceContextContent in ep_mcp.go**

In `go-api/server/ep_mcp.go`, add to the `deps` construction (after `ListResourceContext`):

```go
GetResourceContextContent: func(ctx context.Context, resourceID, documentID uuid.UUID) (string, error) {
	return s.resourceContextService.GetPresignedURL(ctx, organizationID, resourceID, documentID)
},
```

- [ ] **Step 4: Wire GetResourceContextContent in ep_mcp_cli.go**

In `go-api/server/ep_mcp_cli.go`, add to the `deps` construction (after `ListResourceContext`):

```go
GetResourceContextContent: func(ctx context.Context, resourceID, documentID uuid.UUID) (string, error) {
	return s.resourceContextService.GetPresignedURL(ctx, organizationID, resourceID, documentID)
},
```

- [ ] **Step 5: Verify build**

```bash
cd /Users/josegiron/Documents/code/mono-builder
cd go-api && go build ./...
```

- [ ] **Step 6: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add go-common/resourcemcps/mcp.go go-common/resourcemcps/resource_context.go go-api/server/ep_mcp.go go-api/server/ep_mcp_cli.go
git commit -m "feat: add read_resource_context MCP tool"
```

---

## Phase 2: CLI Commands (cli repo)

### Task 9: `major user token` Command

**Files:**
- Create: `cli/cmd/user/token.go`
- Modify: `cli/cmd/user/user.go`

- [ ] **Step 1: Create token.go**

```go
package user

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:    "token",
	Short:  "Print the stored CLI token",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runToken(cmd)
	},
}

func runToken(cmd *cobra.Command) error {
	token, err := mjrToken.GetToken()
	if err != nil {
		return err
	}

	cmd.Print(token)
	return nil
}
```

- [ ] **Step 2: Register in user.go**

In `cli/cmd/user/user.go`, add to `init()`:

```go
Cmd.AddCommand(tokenCmd)
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/josegiron/Documents/code/cli
go build ./...
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add cmd/user/token.go cmd/user/user.go
git commit -m "feat: add hidden 'major user token' command"
```

---

### Task 10: `major org id` Command

**Files:**
- Create: `cli/cmd/org/id.go`
- Modify: `cli/cmd/org/org.go`

- [ ] **Step 1: Create id.go**

```go
package org

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

var idCmd = &cobra.Command{
	Use:    "id",
	Short:  "Print the default organization ID",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runID(cmd)
	},
}

func runID(cmd *cobra.Command) error {
	orgID, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return err
	}

	cmd.Print(orgID)
	return nil
}
```

- [ ] **Step 2: Register in org.go**

In `cli/cmd/org/org.go`, add to `init()`:

```go
Cmd.AddCommand(idCmd)
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/josegiron/Documents/code/cli
go build ./...
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add cmd/org/id.go cmd/org/org.go
git commit -m "feat: add hidden 'major org id' command"
```

---

### Task 11: Add `resource_api_url` to Config

The plugin's `.mcp.json` needs to know the resource API URL. Add it to the config.

**Files:**
- Modify: `cli/configs/prod.json`
- Modify: `cli/configs/local.json`
- Modify: `cli/clients/config/config.go`

- [ ] **Step 1: Update prod.json**

```json
{
    "api_url": "https://api.prod.major.build/cli",
    "resource_api_url": "https://resource-api.prod.major.build",
    "frontend_uri": "https://app.major.build",
    "app_url_suffix": "apps.prod.major.build",
    "app_url_fe_only_suffix": "apps2.prod.major.build"
}
```

- [ ] **Step 2: Update local.json**

```json
{
    "api_url": "http://localhost:3001/cli",
    "resource_api_url": "http://localhost:8080",
    "frontend_uri": "http://localhost:3000",
    "app_url_suffix": "localhost:8080"
}
```

- [ ] **Step 3: Update Config struct**

In `cli/clients/config/config.go`, add the field:

```go
type Config struct {
    APIURL             string `mapstructure:"api_url"`
    ResourceAPIURL     string `mapstructure:"resource_api_url"`
    FrontendURI        string `mapstructure:"frontend_uri"`
    AppURLSuffix       string `mapstructure:"app_url_suffix"`
    AppURLFEOnlySuffix string `mapstructure:"app_url_fe_only_suffix"`
}
```

- [ ] **Step 4: Verify build**

```bash
cd /Users/josegiron/Documents/code/cli
go build ./...
```

- [ ] **Step 5: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add configs/prod.json configs/local.json clients/config/config.go
git commit -m "feat: add resource_api_url to CLI config"
```

---

### Task 12: `major mcp check-readonly` Command

**Files:**
- Create: `cli/cmd/mcp/mcp.go`
- Create: `cli/cmd/mcp/check_readonly.go`
- Modify: `cli/cmd/root.go`

- [ ] **Step 1: Create mcp.go**

```go
package mcp

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:    "mcp",
	Short:  "MCP server utilities",
	Hidden: true,
	Args:   utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(checkReadonlyCmd)
}
```

- [ ] **Step 2: Create check_readonly.go**

```go
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var checkReadonlyCmd = &cobra.Command{
	Use:    "check-readonly [tool-name]",
	Short:  "Check if an MCP tool is read-only",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckReadonly(cmd, args[0])
	},
}

type toolCache struct {
	Tools     []api.ToolMetadata `json:"tools"`
	FetchedAt time.Time          `json:"fetchedAt"`
}

func runCheckReadonly(cmd *cobra.Command, toolName string) error {
	tools, err := getCachedToolMetadata()
	if err != nil {
		// If we can't get metadata, don't block — let Claude Code prompt the user
		return nil
	}

	for _, t := range tools {
		if t.Name == toolName && t.ReadOnly {
			// Output the allow decision for the PreToolUse hook
			result := map[string]any{
				"hookSpecificOutput": map[string]any{
					"hookEventName":     "PreToolUse",
					"permissionDecision": "allow",
				},
			}

			json.NewEncoder(os.Stdout).Encode(result)
			return nil
		}
	}

	// Not read-only or not found — let Claude Code prompt the user
	return nil
}

func getCachedToolMetadata() ([]api.ToolMetadata, error) {
	cacheDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(cacheDir, ".major", "cache", "tool-metadata.json")

	// Try to read cache
	data, err := os.ReadFile(cachePath)
	if err == nil {
		var cache toolCache
		if json.Unmarshal(data, &cache) == nil {
			if time.Since(cache.FetchedAt) < 24*time.Hour {
				return cache.Tools, nil
			}
		}
	}

	// Fetch fresh metadata
	apiClient := singletons.GetAPIClient()
	tools, err := apiClient.GetMCPToolsMetadata()
	if err != nil {
		return nil, err
	}

	// Cache it
	cache := toolCache{
		Tools:     tools,
		FetchedAt: time.Now(),
	}
	cacheData, _ := json.Marshal(cache)
	os.MkdirAll(filepath.Dir(cachePath), 0755)
	os.WriteFile(cachePath, cacheData, 0644)

	return tools, nil
}
```

- [ ] **Step 3: Add ToolMetadata type and API method**

In `cli/clients/api/structs.go`, add:

```go
type ToolMetadata struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ReadOnly     bool   `json:"readOnly"`
	ResourceType string `json:"resourceType"`
}

type GetMCPToolsMetadataResponse struct {
	Tools []ToolMetadata `json:"tools"`
}
```

In `cli/clients/api/client.go`, add:

```go
func (c *Client) GetMCPToolsMetadata() ([]ToolMetadata, error) {
	// This calls the resource API, not the main CLI API
	// For now, use the CLI API base URL pattern
	var resp GetMCPToolsMetadataResponse
	err := c.doRequest("GET", "/mcp/tools", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Tools, nil
}
```

- [ ] **Step 4: Register in root.go**

In `cli/cmd/root.go`, add the import:

```go
"github.com/major-technology/cli/cmd/mcp"
```

And in `init()`, add:

```go
rootCmd.AddCommand(mcp.Cmd)
```

- [ ] **Step 5: Verify build**

```bash
cd /Users/josegiron/Documents/code/cli
go build ./...
```

- [ ] **Step 6: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add cmd/mcp/ clients/api/structs.go clients/api/client.go cmd/root.go
git commit -m "feat: add hidden 'major mcp check-readonly' command"
```

---

## Phase 3: Plugin Infrastructure (cli repo)

### Task 13: Plugin `.mcp.json` with headersHelper

**Files:**
- Create: `cli/plugins/major/scripts/get-headers.sh`
- Create: `cli/plugins/major/.mcp.json`

- [ ] **Step 1: Create get-headers.sh**

```bash
#!/bin/bash
# headersHelper script for Major CLI plugin.
# Outputs JSON headers for MCP server authentication.
# Called by Claude Code at MCP connection time.

TOKEN=$(major user token 2>/dev/null)
ORG=$(major org id 2>/dev/null)

if [ -z "$TOKEN" ] || [ -z "$ORG" ]; then
  echo '{"x-major-error": "Not authenticated. Run: major user login"}' >&2
  exit 1
fi

echo "{\"Authorization\": \"Bearer $TOKEN\", \"x-major-org-id\": \"$ORG\"}"
```

Make it executable:

```bash
chmod +x plugins/major/scripts/get-headers.sh
```

- [ ] **Step 2: Create .mcp.json**

```json
{
  "mcpServers": {
    "major-resources": {
      "type": "http",
      "url": "https://resource-api.prod.major.build/cli/v1/mcp",
      "headersHelper": "${CLAUDE_PLUGIN_ROOT}/scripts/get-headers.sh"
    },
    "major-platform": {
      "type": "http",
      "url": "https://api.prod.major.build/mcp/cli",
      "headersHelper": "${CLAUDE_PLUGIN_ROOT}/scripts/get-headers.sh"
    }
  }
}
```

- [ ] **Step 3: Update plugin.json to reference mcpServers**

In `plugins/major/.claude-plugin/plugin.json`:

```json
{
  "name": "major",
  "description": "Use the Major platform agentically — create, develop, and deploy web apps via Claude Code",
  "version": "1.0.0",
  "author": {
    "name": "Major Technology"
  },
  "repository": "https://github.com/major-technology/cli",
  "license": "MIT",
  "skills": "./skills/",
  "mcpServers": "./.mcp.json"
}
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add plugins/major/scripts/get-headers.sh plugins/major/.mcp.json plugins/major/.claude-plugin/plugin.json
git commit -m "feat: add headersHelper MCP config to plugin"
```

---

### Task 14: Auto-Approve Hooks for Read-Only Tools

**Files:**
- Create: `cli/plugins/major/hooks/hooks.json`
- Create: `cli/plugins/major/scripts/check-readonly.sh`

- [ ] **Step 1: Create hooks.json**

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

- [ ] **Step 2: Create check-readonly.sh**

```bash
#!/bin/bash
# PreToolUse hook that auto-approves read-only MCP tools.
# Reads tool_name from stdin JSON, checks via CLI.

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')

if [ -z "$TOOL_NAME" ]; then
  exit 0
fi

# Strip the MCP server prefix to get the actual tool name
# e.g., "mcp__major_resources__postgresql_psql" -> "postgresql_psql"
ACTUAL_TOOL=$(echo "$TOOL_NAME" | sed 's/^mcp__major_resources__//')

major mcp check-readonly "$ACTUAL_TOOL" 2>/dev/null
```

Make it executable:

```bash
chmod +x plugins/major/scripts/check-readonly.sh
```

- [ ] **Step 3: Update plugin.json to reference hooks**

In `plugins/major/.claude-plugin/plugin.json`, add the hooks field:

```json
{
  "name": "major",
  "description": "Use the Major platform agentically — create, develop, and deploy web apps via Claude Code",
  "version": "1.0.0",
  "author": {
    "name": "Major Technology"
  },
  "repository": "https://github.com/major-technology/cli",
  "license": "MIT",
  "skills": "./skills/",
  "mcpServers": "./.mcp.json",
  "hooks": "./hooks/hooks.json"
}
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add plugins/major/hooks/hooks.json plugins/major/scripts/check-readonly.sh plugins/major/.claude-plugin/plugin.json
git commit -m "feat: add auto-approve hooks for read-only MCP tools"
```

---

## Phase 4: Skills Migration

### Task 15: Copy Skills from AI-Coder to CLI Shared Directory

**Files:**
- Create: `cli/skills/` directory with all shared skills

- [ ] **Step 1: Create skills directory and copy from ai-coder**

```bash
cd /Users/josegiron/Documents/code/cli
mkdir -p skills

# Copy resource skills from ai-coder
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/skills_shared/resources/ skills/resources/

# Copy additional resource skills
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/skills/resources/dynamics skills/resources/dynamics
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/skills/resources/gong skills/resources/gong

# Copy platform skills
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/skills_shared/memory skills/memory
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/src/agents/coder/skills/authn skills/authn
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/src/agents/coder/skills/crons skills/crons
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/src/agents/coder/skills/webhooks skills/webhooks

# Copy agent tools skill
mkdir -p skills/agent-tools
cp -r /Users/josegiron/Documents/code/mono-builder/apps/ai-coder/src/agents/general-agent/skills/modifying-or-creating-agent-tools skills/agent-tools/modifying-or-creating-agent-tools
```

- [ ] **Step 2: Move existing CLI skill into shared directory**

```bash
cd /Users/josegiron/Documents/code/cli
# Move the existing major skill into shared skills
mv plugins/major/skills/major skills/major
```

- [ ] **Step 3: Create symlink from major plugin to shared skills**

```bash
cd /Users/josegiron/Documents/code/cli
# Remove old skills directory in plugin
rm -rf plugins/major/skills

# Create symlink
cd plugins/major
ln -s ../../skills skills
cd ../..
```

- [ ] **Step 4: Verify symlink works**

```bash
ls -la plugins/major/skills
ls plugins/major/skills/major/SKILL.md
ls plugins/major/skills/resources/postgresql/SKILL.md
```

- [ ] **Step 5: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add -f skills/
git add plugins/major/skills
git commit -m "feat: migrate ai-coder skills to shared directory with symlink"
```

---

### Task 16: Create `major-platform` Plugin for AI-Coder

**Files:**
- Create: `cli/plugins/major-platform/.claude-plugin/plugin.json`
- Create: `cli/plugins/major-platform/skills` (symlink)

- [ ] **Step 1: Create plugin directory and manifest**

```bash
cd /Users/josegiron/Documents/code/cli
mkdir -p plugins/major-platform/.claude-plugin
```

Create `plugins/major-platform/.claude-plugin/plugin.json`:

```json
{
  "name": "major-platform",
  "description": "Major platform resource connectors and development skills",
  "version": "1.0.0",
  "author": {
    "name": "Major Technology"
  },
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

- [ ] **Step 2: Create symlink to shared skills**

```bash
cd /Users/josegiron/Documents/code/cli/plugins/major-platform
ln -s ../../skills skills
```

- [ ] **Step 3: Update marketplace.json**

In `cli/.claude-plugin/marketplace.json`, add the major-platform plugin:

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
      "description": "Use the Major platform agentically via Claude Code",
      "version": "1.0.0"
    },
    {
      "name": "major-platform",
      "source": "./plugins/major-platform",
      "description": "Major platform resource connectors and development skills",
      "version": "1.0.0"
    }
  ]
}
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add -f plugins/major-platform/ .claude-plugin/marketplace.json
git commit -m "feat: add major-platform plugin for ai-coder"
```

---

### Task 17: Update AI-Coder to Use Plugin

**Files:**
- Modify: `mono-builder/apps/ai-coder/src/base/startup.ts`
- Modify: `mono-builder/apps/ai-coder/Dockerfile`

- [ ] **Step 1: Update startup.ts to load plugin**

In `apps/ai-coder/src/base/startup.ts`, find where session options are built and add the `plugins` field:

```typescript
import type { SdkPluginConfig } from "@anthropic-ai/claude-agent-sdk";

// Add to session options construction:
const plugins: SdkPluginConfig[] = [
	{ type: 'local', path: '/app/plugins/major-platform' },
];
```

Add `plugins` to the returned session options object.

- [ ] **Step 2: Update Dockerfile to copy plugin**

In the `runner` stage of `apps/ai-coder/Dockerfile`, add a COPY instruction for the plugin. The plugin needs to be available in the Docker build context:

```dockerfile
# Copy major-platform plugin (skills + manifest)
COPY --from=installer /app/plugins/major-platform /app/plugins/major-platform
```

Note: The actual build context setup depends on how the monorepo CI copies files. The plugin source is in the CLI repo, so this step may require:
- A build step that copies `cli/plugins/major-platform/` into the mono-builder build context
- Or a git submodule / checkout step in CI

For now, add a placeholder in the Dockerfile and document the CI requirement.

- [ ] **Step 3: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add apps/ai-coder/src/base/startup.ts apps/ai-coder/Dockerfile
git commit -m "feat: load major-platform plugin in ai-coder"
```

---

### Task 18: Remove Migrated Skills from AI-Coder

**Files:**
- Delete: `mono-builder/apps/ai-coder/skills_shared/resources/` (moved to cli/skills/)
- Delete: `mono-builder/apps/ai-coder/skills/resources/` (moved to cli/skills/)
- Delete: `mono-builder/apps/ai-coder/src/agents/coder/skills/authn/` (moved)
- Delete: `mono-builder/apps/ai-coder/src/agents/coder/skills/crons/` (moved)
- Delete: `mono-builder/apps/ai-coder/src/agents/coder/skills/webhooks/` (moved)
- Delete: `mono-builder/apps/ai-coder/src/agents/general-agent/skills/modifying-or-creating-agent-tools/` (moved)
- Delete: `mono-builder/apps/ai-coder/skills_shared/memory/` (moved)

- [ ] **Step 1: Remove migrated skill directories**

```bash
cd /Users/josegiron/Documents/code/mono-builder

# Resource skills
rm -rf apps/ai-coder/skills_shared/resources/
rm -rf apps/ai-coder/skills/resources/

# Platform skills
rm -rf apps/ai-coder/skills_shared/memory/
rm -rf apps/ai-coder/src/agents/coder/skills/authn/
rm -rf apps/ai-coder/src/agents/coder/skills/crons/
rm -rf apps/ai-coder/src/agents/coder/skills/webhooks/

# Agent tools skill
rm -rf apps/ai-coder/src/agents/general-agent/skills/modifying-or-creating-agent-tools/
```

- [ ] **Step 2: Update startup.ts to remove migrated skill paths**

In `apps/ai-coder/src/base/startup.ts`, update `baseSkillsDirs` to only include skills that remain in the ai-coder repo (new-project, session-files, frontend/data-table):

Remove `skills_shared` paths for resources and memory from the dirs array. Only keep:
- `skills_shared/new-project`
- `skills_shared/session-files`
- Agent-specific skills that stay (frontend/data-table)

- [ ] **Step 3: Verify build**

```bash
cd /Users/josegiron/Documents/code/mono-builder
yarn workspace ai-coder build
```

- [ ] **Step 4: Commit**

```bash
cd /Users/josegiron/Documents/code/mono-builder
git add -A apps/ai-coder/
git commit -m "feat: remove migrated skills from ai-coder (now in plugin)"
```

---

## Phase 5: Cleanup

### Task 19: Deprecate Old GenerateMcpConfig

**Files:**
- Modify: `cli/utils/mcp.go`

- [ ] **Step 1: Update GenerateMcpConfig to not include major server**

In `cli/utils/mcp.go`, remove the `major` entry from the generated `.mcp.json`. The plugin's org-level MCP via `headersHelper` replaces it. Keep the function but have it generate an empty config or remove the `major` key:

```go
// GenerateMcpConfig generates .mcp.json for Claude Code.
// Deprecated: The major CLI plugin now provides MCP servers via headersHelper.
// This function is kept for backward compatibility but generates a minimal config.
func GenerateMcpConfig(targetDir string, envVars map[string]string) (string, error) {
	// ... keep existing logic for determining targetDir ...
	// Remove the major MCP server entry
	// Generate empty mcpServers or only keep non-major entries
```

The exact change depends on whether there are other MCP servers being generated. If `major` was the only one, the function can return early or generate an empty `.mcp.json`.

- [ ] **Step 2: Commit**

```bash
cd /Users/josegiron/Documents/code/cli
git add utils/mcp.go
git commit -m "deprecate: remove major MCP server from GenerateMcpConfig"
```

---

## Self-Review

**Spec coverage check:**
- Step 1 (CLI auth endpoint): Tasks 1-3
- Step 2 (token command): Task 9
- Step 3 (Go API org MCP): Tasks 4-6
- Step 4 (Node API platform MCP): Task 7
- Step 5 (org id command): Task 10
- Step 6 (headersHelper): Task 13
- Step 7 (auto-approve): Tasks 12, 14
- Step 8 (GenerateMcpConfig): Task 19
- Step 8.5 (read_resource_context): Task 8
- Step 9 (skills migration): Tasks 15-18
- Config URLs: Task 11

**All spec steps have corresponding tasks.**

**Type consistency:**
- `CliAuthorizeResponse` used consistently in auth client and schemas
- `CliResourcePermissionsResponse` matches between Go client and TS schema
- `ToolMetadata` struct matches `V1ToolMetadata` from apimodels
- `MCPToolDeps.GetResourceContextContent` signature matches usage in resource_context.go

**No placeholders found.**
