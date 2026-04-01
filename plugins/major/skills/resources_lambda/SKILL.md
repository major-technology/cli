---
name: using-lambda-connector
description: Implements AWS Lambda function invocation and management using generated clients and MCP tools. Use when doing ANYTHING that touches Lambda or AWS Lambda in any way, load this skill.
---

# Major Platform Resource: AWS Lambda

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

- `mcp__resources__lambda_list_functions` — List accessible Lambda functions. Args: `resourceId`, `maxItems?`
- `mcp__resources__lambda_get_function` — Get function config and code location. Args: `resourceId`, `functionName`, `qualifier?`
- `mcp__resources__lambda_invoke` — Invoke a function. Args: `resourceId`, `functionName`, `payload?`, `invocationType?`, `qualifier?`

## TypeScript Client

```typescript
import { lambdaClient } from "./clients";

// invoke(functionName, payload, invocationKey, options?)
const result = await lambdaClient.invoke("my-function", { userId: "123", action: "process" }, "invoke-processor", {
	invocationType: "RequestResponse",
});
if (result.ok) {
	console.log(result.result);
}
```

## Tips

- **Invocation types**: `RequestResponse` (synchronous, default), `Event` (async fire-and-forget), `DryRun` (validate without executing)
- Payload size limit: 6MB for synchronous, 256KB for async invocations
- Use `qualifier` to invoke a specific version or alias (defaults to `$LATEST`)

**Docs**: [AWS Lambda Documentation](https://docs.aws.amazon.com/lambda/)
