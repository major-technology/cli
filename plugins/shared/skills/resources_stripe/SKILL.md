---
name: using-stripe-connector
description: Implements Stripe payment API access for customers, payments, subscriptions, invoices, and balance using generated clients and MCP tools. Use when doing ANYTHING that touches Stripe in any way, load this skill.
---

# Major Platform Resource: Stripe

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Three ways to interact with Stripe:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).
3. **Official Stripe SDK via the HTTP proxy** (Next.js apps): Pass `createProxyFetch` into `Stripe.createFetchHttpClient(...)`. See the **Stripe SDK via the HTTP proxy** section below — preferred when you want full Stripe SDK ergonomics (typed methods, autocomplete, automatic pagination).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `response.ok` before accessing `response.json`.

**Invocation keys must be static strings** — use descriptive literals like `"list-customers"`, never dynamic values like `` `${date}-customers` ``.

---

## MCP Tools

**Tool selection:** For any read-only operation, prefer `stripe_get` (or a specialized `stripe_list_*` / `stripe_get_*` tool) over `stripe_invoke`. Reserve `stripe_invoke` for writes (POST/PUT/DELETE) or endpoints the other tools don't cover.

- `mcp__resources__stripe_get` — **Preferred for all read-only (GET) requests.** Make a GET request to any Stripe API endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__stripe_list_customers` — List customers with optional email filter and cursor pagination. Args: `resourceId`, `email?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_get_customer` — Get a single customer by ID. Args: `resourceId`, `customerId`
- `mcp__resources__stripe_list_payment_intents` — List payment intents with optional filters. Args: `resourceId`, `customer?`, `status?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_get_balance` — Get the current account balance. Args: `resourceId`
- `mcp__resources__stripe_list_subscriptions` — List subscriptions with optional filters. Args: `resourceId`, `customer?`, `status?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_list_invoices` — List invoices with optional filters. Args: `resourceId`, `customer?`, `status?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_invoke` — Make any HTTP request (GET/POST/PUT/DELETE). **Use only for write operations** (POST/PUT/DELETE) or endpoints not covered by `stripe_get` / `stripe_list_*` / `stripe_get_*`. Args: `resourceId`, `method`, `path`, `query?`, `headers?`, `body?`, `timeoutMs?`

## TypeScript Client

The client exposes a single `invoke<T>(method, path, invocationKey, options?)` method. The generic `T` types the parsed JSON response, available directly on `.json`.

### Reading data

```typescript
import { stripeClient } from "./clients";

// List customers filtered by email
const response = await stripeClient.invoke<{
  data: Array<{ id: string; email: string; name: string }>;
  has_more: boolean;
}>("GET", "/v1/customers", "list-customers", {
  query: { limit: ["10"], email: ["jane@example.com"] },
});

if (response.ok) {
  for (const c of response.json.data) {
    console.log(c.id, c.email);
  }
}

// Get account balance
const balance = await stripeClient.invoke<{
  available: Array<{ amount: number; currency: string }>;
  pending: Array<{ amount: number; currency: string }>;
}>("GET", "/v1/balance", "get-balance");

if (balance.ok) {
  console.log("Available:", balance.json.available);
}

// Get a single subscription
const sub = await stripeClient.invoke<{
  id: string;
  status: string;
  current_period_end: number;
}>("GET", "/v1/subscriptions/sub_abc123", "get-subscription");

if (sub.ok) {
  console.log(`Status: ${sub.json.status}`);
}
```

### Writing data

Stripe's v1 API expects `application/x-www-form-urlencoded` bodies for write operations. Use `type: "form"` with a **flat** object whose keys match the Stripe docs. For nested parameters, use Stripe's bracket notation in the key name (e.g. `"metadata[order_id]"`). Do NOT use nested objects or arrays — the platform rejects them. Values must be primitives (string, number, boolean) or null (omitted).

```typescript
// Create a customer — form-encoded body
const created = await stripeClient.invoke<{ id: string }>(
  "POST", "/v1/customers", "create-customer",
  {
    body: {
      type: "form",
      value: {
        email: "jane@example.com",
        name: "Jane Doe",
        "metadata[source]": "onboarding",
      },
    },
  },
);

if (created.ok) {
  console.log("Created:", created.json.id);
}

// Create a payment intent with nested params via bracket keys
const payment = await stripeClient.invoke<{ id: string; client_secret: string }>(
  "POST", "/v1/payment_intents", "create-payment-intent",
  {
    body: {
      type: "form",
      value: {
        amount: 2000,
        currency: "usd",
        customer: "cus_abc123",
        "automatic_payment_methods[enabled]": true,
      },
    },
  },
);

if (payment.ok) {
  console.log("Client secret:", payment.json.client_secret);
}

// Cancel a subscription
await stripeClient.invoke("DELETE", "/v1/subscriptions/sub_xyz789", "cancel-subscription");

// Create a customer session using a preview API version with idempotency
const session = await stripeClient.invoke<{ client_secret: string }>(
  "POST", "/v1/customer_sessions", "create-customer-session",
  {
    headers: {
      "Stripe-Version": "2026-03-25.preview",
      "Idempotency-Key": "session-abc-123",
    },
    body: {
      type: "form",
      value: {
        customer: "cus_abc123",
        "components[pricing_table][enabled]": true,
      },
    },
  },
);
```

### Pagination

```typescript
interface Invoice { id: string; amount_due: number; status: string }

let hasMore = true;
let startingAfter: string | undefined;
const allInvoices: Invoice[] = [];

while (hasMore) {
  const query: Record<string, string[]> = { limit: ["100"], status: ["paid"] };
  if (startingAfter) {
    query.starting_after = [startingAfter];
  }

  const page = await stripeClient.invoke<{ data: Invoice[]; has_more: boolean }>(
    "GET", "/v1/invoices", "list-invoices", { query },
  );

  if (page.ok) {
    allInvoices.push(...page.json.data);
    hasMore = page.json.has_more;
    if (page.json.data.length > 0) {
      startingAfter = page.json.data[page.json.data.length - 1].id;
    }
  } else {
    break;
  }
}
```

## Headers

Pass custom HTTP headers via the `headers` option. Common Stripe headers:

| Header | Purpose |
|--------|---------|
| `Stripe-Version` | Pin a specific API version (e.g. `"2026-03-25.preview"` for preview features). |
| `Idempotency-Key` | Ensure POST requests are idempotent — Stripe deduplicates by this key. |
| `Stripe-Account` | Make requests on behalf of a connected account (Stripe Connect). |

**Protected headers:** `Authorization` and `Content-Type` are managed by the connector and **cannot** be set via `headers`. Attempting to do so returns an error.

```typescript
await stripeClient.invoke("GET", "/v1/balance", "get-connected-balance", {
  headers: { "Stripe-Account": "acct_connected123" },
});
```

## Body Types

The `body` option supports multiple types:

| Type | Content-Type | When to use |
|------|-------------|-------------|
| `"form"` | `application/x-www-form-urlencoded` | **Default for Stripe writes.** All v1 POST/PUT/PATCH endpoints expect form encoding. Use flat keys with bracket notation for nested params. |
| `"json"` | `application/json` | v2 API endpoints that accept JSON. |
| `"text"` | `text/plain` | Rarely needed. |
| `"bytes"` | Custom (set via `contentType`) | Binary payloads (base64-encoded in `base64` field). |

### Form body rules
- **Flat keys only.** Use Stripe's bracket notation for nested params: `"metadata[order_id]"`, `"items[0][price]"`.
- **No nested objects or arrays** in the value — the platform rejects them with a clear error. This avoids hidden flattening ambiguity.
- **Primitive values:** `string`, `number` (converted to decimal string), `boolean` (converted to `"true"` / `"false"`).
- **Empty string** is preserved (sends `key=`).
- **Null values** are omitted from the encoded body.

## Stripe SDK via the HTTP proxy

For Next.js apps you can also use the official `stripe` npm package and route every call through the Major HTTP proxy. The proxy injects the secret key, so you never touch credentials in app code. See [using-http-proxy](../http-proxy/SKILL.md) for the proxy reference.

**Setup:**

```bash
pnpm add stripe @major-tech/resource-client
```

**Usage (Server Components, Route Handlers, Server Actions):**

```typescript
import Stripe from "stripe";
import { createProxyFetch } from "@major-tech/resource-client/next";

const proxyFetch = createProxyFetch({
  baseUrl: process.env.MAJOR_API_BASE_URL!,
  resourceId: process.env.STRIPE_RESOURCE_ID!,
  majorJwtToken: process.env.MAJOR_JWT_TOKEN!,
});

const stripe = new Stripe("sk_unused_proxy_injects_real_key", {
  httpClient: Stripe.createFetchHttpClient(proxyFetch),
});

// Use the SDK normally — every request flows through the proxy
const customers = await stripe.customers.list({ limit: 10 });
const intent = await stripe.paymentIntents.create({
  amount: 2000,
  currency: "usd",
  customer: "cus_abc123",
});
```

**Rules:**

- `MAJOR_API_BASE_URL` and `MAJOR_JWT_TOKEN` are platform-managed env vars — assume they're set, don't ask the user to provide them. `STRIPE_RESOURCE_ID` is the UUID of the connected Stripe resource.
- The placeholder `"sk_unused_..."` constructor key is never sent — the proxy strips `Authorization` and injects the real secret key.
- Do NOT set the `Authorization` header anywhere. The proxy will strip it.
- This only works server-side (Server Components, Route Handlers, Server Actions). The `/next` subpath uses `next/headers`, which is not available in client components.
- Use a static `STRIPE_RESOURCE_ID` string literal.
- API version pinning still works — pass `apiVersion: "2026-03-25.preview"` in the Stripe constructor or `Stripe-Version` per-request via the SDK's standard mechanisms.

## Tips

- **Use `type: "form"` for Stripe v1 writes.** Stripe's v1 API natively expects form-encoded bodies. While `type: "json"` also works (Stripe accepts both), `type: "form"` is the canonical encoding documented by Stripe.
- **Paths must start with `/v1/` or `/v2/`.** The platform validates paths and rejects absolute URLs, protocol-relative paths, and paths outside `/v1/` or `/v2/`.
- **Pagination**: Stripe uses cursor-based pagination. Pass `starting_after` with the last object's ID to get the next page. Check `has_more` in the response.
- **All list endpoints** support `limit` (default 10, max 100).
- **Expand related objects**: Use the `expand[]` query param to inline related objects instead of just their IDs.
- **Test vs live keys**: Test mode keys start with `sk_test_`, live mode with `sk_live_`. The connector works with either — just provide the right key.
- **Stripe API versioning**: No `Stripe-Version` header is sent by default, so Stripe uses your account's default API version. Pass `headers: { "Stripe-Version": "..." }` to pin a specific version.
- **Common v1 endpoints**: `/v1/customers`, `/v1/payment_intents`, `/v1/subscriptions`, `/v1/invoices`, `/v1/charges`, `/v1/balance`, `/v1/refunds`, `/v1/products`, `/v1/prices`

**Docs**: [Stripe API Reference](https://docs.stripe.com/api)
