---
name: using-docusign-connector
description: Implements DocuSign eSignature data access via Pipedream Connect using MCP tools. Use when doing ANYTHING that touches DocuSign envelopes, signing, or templates.
---

# Major Platform Resource: DocuSign

DocuSign access is brokered through **Pipedream Connect's API proxy**. Major holds no DocuSign secrets — Pipedream owns the OAuth tokens. Outbound URLs must point to `account.docusign.com` or `*.docusign.{net,com}`; anything else is rejected with a 400.

## Common: Interacting with Resources

**Security**: Never connect directly to DocuSign. Never use credentials in code. Always use the MCP tools below.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__docusign_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Direct invoke from app code** (for typed app integrations): Until a typed `DocusignResourceClient` is published in `@major-tech/resource-client`, app code can hit the resource invoke endpoint directly with a `{type: "api", subtype: "docusign", method, url, ...}` payload. Prefer the MCP tools when prototyping.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** — use descriptive literals like `"list-sent-envelopes"`, never dynamic values like `` `${date}-envelopes` ``.

---

## Discovery-first

DocuSign URLs are scoped to an account_id (`/v2.1/accounts/{accountId}/...`), and `base_uri` varies per account (different DocuSign data centers). **Major does NOT store these** — the caller is responsible for assembling URLs. Always call `docusign_get_userinfo` first to discover them:

```
mcp__resources__docusign_get_userinfo(resourceId)
```

The response includes:

```json
{
    "accounts": [
        { "account_id": "abc-123", "account_name": "Acme", "base_uri": "https://demo.docusign.net", "is_default": true },
        {
            "account_id": "def-456",
            "account_name": "Acme Sandbox",
            "base_uri": "https://demo.docusign.net",
            "is_default": false
        }
    ]
}
```

Pick the default account (or whichever one the user means) and pass `base_uri` and `account_id` into every subsequent tool.

## MCP Tools

- `mcp__resources__docusign_get_userinfo` — Discover the DocuSign accounts the connected user can access. Returns `base_uri` + `account_id` required by every other tool. Args: `resourceId`.
- `mcp__resources__docusign_list_envelopes` — List envelopes for an account, with optional status / date / search filters. Args: `resourceId`, `baseUri`, `accountId`, `status?`, `fromDate?`, `toDate?`, `searchText?`.
- `mcp__resources__docusign_get_envelope` — Fetch a single envelope's metadata + status. Args: `resourceId`, `baseUri`, `accountId`, `envelopeId`.
- `mcp__resources__docusign_create_envelope` — Create + send a new envelope. Args: `resourceId`, `baseUri`, `accountId`, `envelopeDefinition`.
- `mcp__resources__docusign_download_document` — Download a signed/in-progress PDF as base64 bytes. Use `documentId="combined"` for the merged PDF of all documents in the envelope. Args: `resourceId`, `baseUri`, `accountId`, `envelopeId`, `documentId`.
- `mcp__resources__docusign_get` — Read-only DocuSign HTTP GET escape hatch for any DocuSign endpoint not covered by a typed tool. Args: `resourceId`, `url`, `query?`, `headers?`.
- `mcp__resources__docusign_invoke` — Full DocuSign HTTP escape hatch (any method). Use for endpoints not covered by typed tools, especially write operations like voiding envelopes or updating templates. Args: `resourceId`, `method`, `url`, `query?`, `headers?`, `body?`.

## App code (direct invoke)

Until a typed `DocusignResourceClient` ships in `@major-tech/resource-client`, app code calls DocuSign through the resource invoke endpoint with a Pipedream-shaped payload. Always start with `/oauth/userinfo` to discover account state.

```typescript
async function callDocusign(method: string, url: string, invocationKey: string, body?: unknown) {
    const response = await fetch(`/api/resources/${RESOURCE_ID}/invoke`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
            invocationKey,
            payload: {
                type: "api",
                subtype: "docusign",
                method,
                url,
                ...(body ? { body: { type: "json", value: body } } : {}),
            },
        }),
    });

    return response.json();
}

// 1. Discover account
const userinfo = await callDocusign("GET", "https://account.docusign.com/oauth/userinfo", "discover-docusign-account");
const acct = userinfo.result.body.value.accounts.find((a) => a.is_default);

// 2. List envelopes
const envelopes = await callDocusign(
    "GET",
    `${acct.base_uri}/restapi/v2.1/accounts/${acct.account_id}/envelopes?from_date=2026-01-01&status=sent`,
    "list-recent-envelopes",
);

// 3. Download a signed PDF (response body is base64 bytes)
const pdf = await callDocusign(
    "GET",
    `${acct.base_uri}/restapi/v2.1/accounts/${acct.account_id}/envelopes/${envelopes.result.body.value.envelopes[0].envelopeId}/documents/combined`,
    "download-signed-pdf",
);
if (pdf.result.body.kind === "bytes") {
    const fileBytes = Buffer.from(pdf.result.body.base64, "base64");
}
```

## Tips

- **Always discover first.** Call `docusign_get_userinfo` before any other tool. Major does not store `account_id` or `base_uri`; the caller passes them per request.
- **Allowed URLs**: `account.docusign.com`, `*.docusign.net`, `*.docusign.com` over HTTPS only. Anything else returns 400.
- **Status values for envelopes**: `created`, `sent`, `delivered`, `signed`, `completed`, `declined`, `voided`. Filter with `status=sent,delivered` (comma-separated for multiple).
- **Date filters**: `from_date` and `to_date` accept ISO 8601 dates (`2026-01-01`).
- **Document downloads return bytes**: response body comes back as `{ kind: "bytes", base64: "...", contentType: "application/pdf" }`. Decode the base64 to get the PDF. The combined PDF (`documentId="combined"`) merges all documents in the envelope.
- **Pipedream proxy size cap**: Document downloads larger than ~6MB may fail at the Pipedream proxy layer. For very large signed PDFs, consider splitting envelope documents instead of using `combined`.
- **Multi-account users**: When `userinfo.accounts` has more than one entry, use the one the user means rather than auto-picking the default. Confirm with the user if ambiguous.
- **Reconnect**: If `TestConnection` fails or calls return 401 from upstream, the user needs to reconnect via the DocuSign panel — Major automatically reuses the same `external_user_id` on reconnect, so Pipedream replaces the underlying account record.
- **Response shape**: Every typed tool returns `{ kind: "api", status: number, body: { kind: "json"|"text"|"bytes", ... } }`. Always check `body.kind` before reading `value` vs `base64`.

**Docs**: [DocuSign eSignature API Reference](https://developers.docusign.com/docs/esign-rest-api/reference/)
