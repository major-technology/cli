---
name: using-webhooks
description: Explains how webhook access works on deployed Major apps and how to implement webhook endpoints. Use when the user mentions webhooks, needs to receive external HTTP callbacks, or is integrating with services that send webhooks (e.g., Stripe, GitHub, Twilio).
---

# Major Platform: Webhooks

## Overview

Major apps are protected by an authentication gateway by default. Enabling **Webhook Access** allows unauthenticated HTTP traffic to reach routes matching `/api/webhook/*` on **deployed apps only**. This is required for receiving callbacks from external services like Stripe, GitHub, Twilio, etc.

**Important:** Webhook access simply allows traffic through — it does NOT perform any authentication or verification. If the webhook provider supports authentication (e.g., Stripe signature verification, GitHub webhook secrets, Twilio request validation), the application **MUST** implement its own verification at the application layer.

---

## Two Options

- **Webhook token (preferred):** for any sender that can set a URL or a header. This includes Zapier, internal services, scripts, and signing senders such as Stripe and GitHub, where the token adds a second layer. The sender sends the token as `x-major-jwt: <token>` or as `?major_token=<token>`. The token works only on the app it was created for, on both the deployed app and the preview.
- **Unauthenticated toggle:** for senders that cannot carry a token. The app must verify the provider signature, as described below.

### Where tokens are created

Major dashboard > open the app > Settings tab > Webhook Access > Webhook tokens > Create. Only app admins can create tokens. The agent never creates or sees a token. Tell the user to create one and to paste the URL (`https://<app-host>/api/webhook/<your-route>?major_token=<token>`) into the sender.

### App code with tokens

The request arrives as the token's creator in `x-major-user-id`. The app must not log full request URLs, because the query can contain `major_token`.

---

## Checking & Enabling Webhooks

Run `major app info --json` from the app workspace to check the current app status, including `webhooksEnabled`. In a hosted sandbox, run it through `mcp__plugin_major_major__sandbox_bash`.

If webhooks are not enabled, instruct the user to enable them from the **Major dashboard → App Settings → Webhook Access** toggle.

**Note:** If the app is already public (`isPublic: true`), webhook access is not needed — all unauthenticated traffic is already allowed.

---

## Implementing Webhook Endpoints

Create your webhook routes under the `/api/webhook/` path. Only routes matching this pattern will bypass authentication when webhook access is enabled. The backend's app already has `.basePath("/api")`, so a route at `/webhook/stripe` serves `/api/webhook/stripe`.

```typescript
// backend/src/routes/webhooks.ts
import { createRoute, z, type OpenAPIHono } from "@hono/zod-openapi";
import Stripe from "stripe";

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!);
const webhookSecret = process.env.STRIPE_WEBHOOK_SECRET!;

// No request body schema: the signature check needs the raw body, so nothing may parse it first.
const stripeWebhookRoute = createRoute({
	method: "post",
	path: "/webhook/stripe",
	responses: {
		200: {
			description: "The event was verified and handled",
			content: { "application/json": { schema: z.object({ received: z.boolean() }) } },
		},
		400: {
			description: "The signature did not verify",
			content: { "application/json": { schema: z.object({ error: z.string() }) } },
		},
	},
});

export function registerWebhookRoutes(app: OpenAPIHono) {
	app.openapi(stripeWebhookRoute, async (c) => {
		const body = await c.req.text();
		const signature = c.req.header("stripe-signature") ?? "";

		// CRITICAL: Verify the webhook signature at the application layer
		let event: Stripe.Event;
		try {
			event = stripe.webhooks.constructEvent(body, signature, webhookSecret);
		} catch {
			return c.json({ error: "Invalid signature" }, 400);
		}

		// Handle the verified event
		switch (event.type) {
			case "checkout.session.completed":
				// Process the event
				break;
		}

		return c.json({ received: true }, 200);
	});
}
```

Register it in `backend/src/app.ts` (`registerWebhookRoutes(app);`) and install `stripe` in the backend (`pnpm --filter ./backend add stripe`). In an older Next.js app, the same handler goes in `app/api/webhook/stripe/route.ts` as `export async function POST(request: Request)`, reading the body with `await request.text()`.

---

## Tips

- **Deployed apps only** — Webhook bypass only applies to deployed applications, not coding sessions
- **Route pattern** — Only `/api/webhook/*` routes are bypassed. All other routes still require authentication
- **Always verify signatures** — Since Major does not authenticate webhook requests, you MUST verify request authenticity yourself using provider-specific mechanisms (signatures, shared secrets, etc.)
- **Public apps** — If the app is already public, the webhook toggle is irrelevant since all routes are already accessible without authentication
- **Environment variables** — Store webhook secrets (e.g., `STRIPE_WEBHOOK_SECRET`, `GITHUB_WEBHOOK_SECRET`) as environment variables in the Major dashboard
