---
name: auth-cognito
description: Use when building or reviewing sign-in for an app composed with the auth-cognito plugin (auth@1) - who owns a record, which routes need a person, a group-gated action, the sign-in, sign-up, confirm or reset screens, the route guard, or signing in as a test identity in a local run. The identity seam, the session module, the guard and the sign-in calls are already in the repository and managed; the build writes authorisation and edits the seeded screens.
metadata:
  capability: auth
---

# Cognito auth (auth-cognito)

A first-time visitor can register from the app, confirm their email, sign in and reach a protected
route. All of that is in the repository: read it, then extend it. The plugin declares
`services.auth` (provider `cognito`, posture `private`, `sign_up: open`) in `albitor.yaml`; the
deploy target realises the user pool and hands the function `AUTH_USER_POOL_ID`, `AUTH_CLIENT_ID`
and `AWS_REGION`. There is no Terraform and no pool id in the repository.

## What is managed (never edit)

| File | What it does |
| --- | --- |
| `api/internal/identity/identity.go` | `identity.Owner(r)`, `identity.FromContext(ctx)`, `Identity{Sub, Email, Groups}` and `InGroup`. The one identity API a handler uses |
| `api/internal/shell/identity_verify.go` | `IdentityVerify` (the `api.middleware` entry) refuses every `/api/` request without a verified token except `/api/health`, `/api/auth/*` and an unknown path; `IdentityRoutes` serves `GET /api/me` and `POST /api/auth/*` |
| `api/internal/shell/identity_token.go` | The RS256 check against the pool's JWKS: signature, `iss`, `token_use`, client, expiry. Standard library only |
| `api/internal/shell/identity_cognito.go` | Where the pool comes from, and the sign-in, sign-up, confirm, resend, forgot, reset and sign-out calls. The session is two HttpOnly, Secure, SameSite=Strict cookies |
| `web/src/lib/auth/session.ts` | `getSession()`, which asks `GET /api/me`. Provider-independent: the guard and the screens both read it |
| `web/src/lib/auth/guard.ts` | `requireSession`, the `web.route-guard` entry. Sends a signed-out person to `/sign-in?next=...` |
| `web/src/lib/auth/client.ts` | `signIn`, `signUp`, `confirm`, `resend`, `forgot`, `reset`, `signOut`. They return `{ ok: false, code, message }` rather than throw |

The browser never talks to Cognito (the page's CSP allows its own origin only) and no token is
readable by page script. A script or test client may send `Authorization: Bearer <token>` instead.

## What the build writes

- **Authorisation.** Take the owner from `identity.Owner(r)`, put it in the partition key, and
  gate privileged routes on `id.InGroup("admin")` after `identity.FromContext`. Never take the
  owner from a body or a query parameter.
- **The screens** in `web/src/routes/(auth)/` are seeded once and yours: restyle and reword them,
  add sign-up fields (and the field in `client.ts`'s call), but **keep the `data-testid` values**
  (`sign-in-form`, `sign-in-email`, `sign-in-password`, `sign-in-button`, and the sign-up, confirm and
  reset ones): the journey harness signs in through them. `sign-out/nav.ts` puts "Sign out" in the
  nav; delete it to take it out.
- **Public pages.** `PUBLIC_PATHS` in `guard.ts` is managed. A page that must be public (a landing
  page, say) means ejecting `guard.ts` and adding the path: say why in the delivery record. The
  guard only shapes what the browser shows; the API enforces access.
- **Anything the brief adds** (MFA, social sign-in, invite-only) is a change to `services.auth` in
  `albitor.yaml` and, where the API must change, an ejected file. `sign_up` is `open` or
  `invite_only`; a challenge the screens do not handle answers `challenge_unsupported`.

## Rules

- **Do not weaken the pool.** The pool is the deploy target's, from `services.auth`: no Lambda
  trigger, no auto-confirm, at least 12 characters. The floor gates refuse all of them.
- **No user-pool admin action on the API's role.** Seed demo accounts with the deploy identity.
- **Fail closed.** With no pool configured on anything that is not a local run, every `/api/`
  route answers 503. Do not add a fallback.
- **Only one spec registers.** A sign-up sends an email, and the default Cognito sender allows 50 a
  day per AWS account. Other specs use the suite's seeded session or a local identity.
- **No Cognito Hosted UI.** The screens are the app's own.

## Local run

`ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1 go run .` starts api-go's local issuer. `IdentityVerify`
then trusts it and nothing else (never beside a named pool or on a deployed function), and the
sign-in calls answer 501 `local`. A spec signs in as `alice`, `bob` or `admin` through
`web/tests/helpers/local-auth.ts` (`signInAs`); the journey harness does this for every journey
test. With `ALBITOR_LOCAL=1` and no issuer, every request acts for `identity.AnonymousOwner`.

## Verification

`web/tests/journeys/sign-in.spec.ts` (seeded, yours to extend) proves a signed-out visitor is sent
to sign-in, the screens render and pass axe, and a local identity reaches the app. The universal
`api-auth` floor still applies: declared non-public endpoints return `401` unauthenticated.
