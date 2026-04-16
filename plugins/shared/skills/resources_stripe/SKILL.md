---
name: using-stripe-connector
description: Implements Stripe payment API access for customers, payments, subscriptions, invoices, and balance using generated clients and MCP tools. Use when doing ANYTHING that touches Stripe in any way, load this skill.
---

# Major Platform Resource: Stripe

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `response.ok` before accessing `response.json`.

**Invocation keys must be static strings** — use descriptive literals like `"list-customers"`, never dynamic values like `` `${date}-customers` ``.

---

## MCP Tools

- `mcp__resources__stripe_get` — Make a GET request to any Stripe API endpoint. Args: `resourceId`, `path`, `query?`
- `mcp__resources__stripe_list_customers` — List customers with optional email filter and cursor pagination. Args: `resourceId`, `email?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_get_customer` — Get a single customer by ID. Args: `resourceId`, `customerId`
- `mcp__resources__stripe_list_payment_intents` — List payment intents with optional filters. Args: `resourceId`, `customer?`, `status?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_get_balance` — Get the current account balance. Args: `resourceId`
- `mcp__resources__stripe_list_subscriptions` — List subscriptions with optional filters. Args: `resourceId`, `customer?`, `status?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_list_invoices` — List invoices with optional filters. Args: `resourceId`, `customer?`, `status?`, `limit?`, `startingAfter?`
- `mcp__resources__stripe_invoke` — Make any HTTP request (GET/POST/PUT/DELETE) for write operations. Args: `resourceId`, `method`, `path`, `query?`, `body?`, `timeoutMs?`

## TypeScript Client

The client exposes a single `invoke<T>(method, path, invocationKey, options?)` method. The generic `T` types the parsed JSON response, available directly on `.json`. All request bodies are sent as JSON.

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
```

### Writing data

```typescript
// Create a customer
const created = await stripeClient.invoke<{ id: string }>(
  "POST", "/v1/customers", "create-customer",
  { body: { type: "json", value: { email: "jane@example.com", name: "Jane Doe" } } },
);

if (created.ok) {
  console.log("Created:", created.json.id);
}

// Create a payment intent
const payment = await stripeClient.invoke<{ id: string; client_secret: string }>(
  "POST", "/v1/payment_intents", "create-payment-intent",
  { body: { type: "json", value: { amount: 2000, currency: "usd", customer: "cus_abc123" } } },
);

if (payment.ok) {
  console.log("Client secret:", payment.json.client_secret);
}
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

## Tips

- **All request bodies are JSON.** The platform sends `Content-Type: application/json` for all Stripe requests.
- **Pagination**: Stripe uses cursor-based pagination. Pass `starting_after` with the last object's ID to get the next page. Check `has_more` in the response.
- **All list endpoints** support `limit` (default 10, max 100).
- **Expand related objects**: Use the `expand[]` query param to inline related objects instead of just their IDs.
- **Test vs live keys**: Test mode keys start with `sk_test_`, live mode with `sk_live_`. The connector works with either.
- **Common endpoints**: `/v1/customers`, `/v1/payment_intents`, `/v1/subscriptions`, `/v1/invoices`, `/v1/charges`, `/v1/balance`, `/v1/refunds`, `/v1/products`, `/v1/prices`

**Docs**: [Stripe API Reference](https://docs.stripe.com/api)
