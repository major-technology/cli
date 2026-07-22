---
name: using-ecr-connector
description: Lists AWS ECR repositories and image tags using MCP tools and generated TypeScript clients. Use when doing ANYTHING that touches ECR or container registries.
---

# Major Platform Resource: AWS ECR

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"fetch-user-orders"`, never dynamic values like `` `${date}-records` ``.

---

## MCP Tools

- `mcp__resources__ecr_list_repositories` — List all ECR repositories. Args: `resourceId`, `maxResults?`, `nextToken?`
- `mcp__resources__ecr_list_images` — List image IDs in a repository. Args: `resourceId`, `repositoryName`, `maxResults?`, `nextToken?`
- `mcp__resources__ecr_describe_images` — Get detailed image info (tags, digest, size, pushed date). Args: `resourceId`, `repositoryName`, `imageIds?`, `maxResults?`, `nextToken?`

## TypeScript Client

```typescript
import { ecrClient } from "./clients";

// List all repositories
const repos = await ecrClient.invoke("ListRepositories", {}, "list-repos");
if (repos.ok) {
	const repositories = repos.result.data;
}

// List images in a repository
const images = await ecrClient.invoke("ListImages", { repositoryName: "my-app" }, "list-images");

// Get detailed image info including tags
const details = await ecrClient.invoke(
	"DescribeImages",
	{
		repositoryName: "my-app",
		imageIds: [{ imageTag: "latest" }],
	},
	"describe-images",
);
```

## Tips

- ECR is **read-only** — this connector lists repos and image metadata only
- Use `DescribeImages` to get full detail: tags, digest, size, vulnerability scan status, push timestamp
- `ListImages` returns image IDs (tag + digest); `DescribeImages` returns full metadata
- For pagination, pass the `nextToken` from a previous response
- Default region is configured on the connector; it determines which ECR registry is queried

**Docs**: [Amazon ECR Documentation](https://docs.aws.amazon.com/ecr/)
