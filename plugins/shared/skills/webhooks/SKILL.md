---
name: using-webhooks
description: Explains how webhook access works on deployed Major apps and how to implement webhook endpoints. Use when the user mentions webhooks, needs to receive external HTTP callbacks, or is integrating with services that send webhooks (e.g., Stripe, GitHub, Twilio).
---

# Major Platform: Webhooks

## Overview

Major apps are protected by an authentication gateway by default. Enabling **Webhook Access** allows unauthenticated HTTP traffic to reach routes matching `/api/webhook/*` on **deployed apps only**. This is required for receiving callbacks from external services like Stripe, GitHub, Twilio, etc.

**Important:** Webhook access simply allows traffic through — it does NOT perform any authentication or verification. If the webhook provider supports authentication (e.g., Stripe signature verification, GitHub webhook secrets, Twilio request validation), the application **MUST** implement its own verification at the application layer.

---

## Checking & Enabling Webhooks

Use the `mcp__major-platform__get_app_status` tool to check the current app status, including `webhooksEnabled`.

If webhooks are not enabled, instruct the user to enable them from the **Major dashboard → App Settings → Webhook Access** toggle.

**Note:** If the app is already public (`isPublic: true`), webhook access is not needed — all unauthenticated traffic is already allowed.

---

## Implementing Webhook Endpoints

Create your webhook routes under the `/api/webhook/` path. Only routes matching this pattern will bypass authentication when webhook access is enabled.

```typescript
// app/api/webhook/stripe/route.ts
import { NextRequest, NextResponse } from "next/server";
import Stripe from "stripe";

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!);
const webhookSecret = process.env.STRIPE_WEBHOOK_SECRET!;

export async function POST(request: NextRequest) {
	const body = await request.text();
	const signature = request.headers.get("stripe-signature")!;

	// CRITICAL: Verify the webhook signature at the application layer
	let event: Stripe.Event;
	try {
		event = stripe.webhooks.constructEvent(body, signature, webhookSecret);
	} catch (err) {
		return NextResponse.json({ error: "Invalid signature" }, { status: 400 });
	}

	// Handle the verified event
	switch (event.type) {
		case "checkout.session.completed":
			// Process the event
			break;
	}

	return NextResponse.json({ received: true });
}
```

---

## Tips

- **Deployed apps only** — Webhook bypass only applies to deployed applications, not coding sessions
- **Route pattern** — Only `/api/webhook/*` routes are bypassed. All other routes still require authentication
- **Always verify signatures** — Since Major does not authenticate webhook requests, you MUST verify request authenticity yourself using provider-specific mechanisms (signatures, shared secrets, etc.)
- **Public apps** — If the app is already public, the webhook toggle is irrelevant since all routes are already accessible without authentication
- **Environment variables** — Store webhook secrets (e.g., `STRIPE_WEBHOOK_SECRET`, `GITHUB_WEBHOOK_SECRET`) as environment variables in the Major dashboard
