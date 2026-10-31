---
name: auth-none
description: Use when an app is composed with the auth-none plugin (auth@1) - an app with no sign-in, such as a public read-only site, an API-less shell or a prototype - and you need to know who owns a record, or what changes if sign-in is added later. Every request acts for one fixed owner.
metadata:
  capability: auth
---

# No auth (auth-none)

This app has no sign-in. Anyone who can reach it can use all of it, and that is the choice that was
made: say so in the delivery record, and do not describe the app as private.

## What it gives you

`api/internal/identity` (seeded once, yours to edit), with the same API as the auth-cognito plugin's:

| Call | Answers |
| --- | --- |
| `identity.Owner(r)` | Always `identity.FixedOwner` (`"owner"`) |
| `identity.FromContext(ctx)` | Nothing: no middleware puts an identity on the request |

There are no screens, no route guard, no middleware, no `GET /api/me` and no `services.auth` in
`albitor.yaml`. It requires nothing, so a web-only profile (no API) composes it and gets no files at
all.

## Roles

There are none. `web/tests/auth/provider.ts` (seeded with the Playwright harness) is this plugin's
`auth@2` provider: `auth.roles()` is always empty, and `seed` and `signInAs` refuse, so no spec can
pretend to be someone. An app that needs roles needs a real auth plugin.

## Writing handlers

Use `identity.Owner(r)` for the owner exactly as you would under a real sign-in: put it in the
partition key, never take it from a body or a query parameter. Then adding sign-in later (replacing
this plugin with auth-cognito) changes who the owner is and nothing in the handlers. Do not
hard-code `"owner"`.

Do not add a login of your own on top of this plugin (a shared password, a header check): that is
the auth socket's job, and a second mechanism is the one thing a later swap cannot absorb.
