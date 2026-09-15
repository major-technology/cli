---
name: using-authn
description: Implements AuthN (identity) operations. Use whenever building anything that requires AuthN
---

To get a user's identity, use @lib/auth.ts. If the user is logged in, the endpoint will return the user's email, userId, and name.

All AuthZ should be handled by the user's application. This function should be purely used for identifying the current user that is logged in. There should be no other way of determining the user's identity.
