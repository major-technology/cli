---
name: using-authn
description: Implements AuthN (identity) operations. Use whenever building anything that requires AuthN
---

To get a user's identity, call `getCurrentUser()` from `backend/src/lib/auth.ts` in backend code (`lib/auth.ts` in an older Next.js app). If the user is logged in, it returns the user's email, userId, and name. The frontend gets the user from the template's `GET /api/me` route, never by reading a token itself.

All AuthZ should be handled by the user's application. This function should be purely used for identifying the current user that is logged in. There should be no other way of determining the user's identity.
