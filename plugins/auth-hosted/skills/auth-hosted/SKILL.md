---
name: auth-hosted
description: Use when building or reviewing sign-in for a hosted-lane app composed with the auth-hosted plugin (auth@1) - who owns a record, which routes need a person, a group-gated action, the sign-in redirect, sign-out, the account and deletion screens, the sign-up domain refusal, the route guard, or acting as a test identity in a local run. The identity seam, the session module, the guard and the hosted sign-in calls are already in the repository and managed; the build writes authorisation and edits the seeded screens.
metadata:
  capability: auth
---

# Hosted sign-in (auth-hosted)

A first-time visitor is sent to the app's Cognito hosted page, signs up or in there, and comes back
signed in. All of the app's side is in the repository: read it, then extend it. The plugin declares
`services.auth` (provider `cognito`, posture `private`, `sign_up: open`) in `albitor.yaml`; the hosted
deploy realises the pool and hands the function `AUTH_ISSUER`, `AUTH_CLIENT_ID`, `AUTH_JWKS_URL`,
`AUTH_POSTURE`, `AUTH_LOGIN_URL` and `AUTH_LOGOUT_URL`. There is no Terraform and no pool id in the
repository.

## What is managed (never edit)

| File | What it does |
| --- | --- |
| `api/internal/identity/identity.go` | `identity.Owner(r)`, `identity.FromContext(ctx)`, `Identity{Sub, Email, Groups}` and `InGroup`. The one identity API a handler uses |
| `api/internal/shell/identity_verify.go` | `IdentityVerify` (the `api.middleware` entry) refuses every `/api/` request without a verified token except `/api/health`, `/api/auth/*` and an unknown path; `IdentityRoutes` serves `GET /api/me` and `/api/auth/*` |
| `api/internal/shell/identity_token.go` | The RS256 check against the pool's JWKS: signature, `iss`, `token_use`, client, expiry. Standard library only |
| `api/internal/shell/identity_hosted.go` | The environment, `GET /api/auth/login` and `/callback` (authorization code with PKCE), refresh, `POST /api/auth/sign-out` and `/delete`, the `SIGNUP_DOMAINS` rule. The session is `__Host-` HttpOnly, Secure, SameSite=Strict cookies |
| `web/src/lib/auth/session.ts` | `getSession()`, which asks `GET /api/me`. Provider-independent: the guard and the screens both read it |
| `web/src/lib/auth/guard.ts` | `requireSession`, the `web.route-guard` entry. Sends a signed-out person to `/sign-in?next=...` |
| `web/src/lib/auth/client.ts` | `loginUrl`, `signOut`, `deleteAccount`. They return `{ ok: false, code, message }` rather than throw |

The browser never talks to Cognito's API and no token is readable by page script. A script or test
client may send `Authorization: Bearer <token>` instead.

## What the build writes

- **Authorisation.** Take the owner from `identity.Owner(r)`, put it in the partition key, and gate
  privileged routes on `id.InGroup("admin")` after `identity.FromContext`. Never take the owner from a
  body or a query parameter.
- **The screens** in `web/src/routes/(auth)/` are seeded once and yours: restyle and reword them. Keep
  the `data-testid` values and the headings the journey spec reads. There is **no password field**: do
  not add a sign-in form, a sign-up screen or a reset screen, because Cognito's pages do those.
- **Who may sign in.** `SIGNUP_DOMAINS` (comma-separated `example.org` or `*.example.org`) limits sign-in
  by email domain. A refused person lands on `/not-allowed`; reword that page and say who to contact. The rule and `email_verified` are applied to every ID token the API takes (callback, refresh, every request), so a token from a hand-run flow gets nowhere; the gate accepts ID tokens only.
  The platform's seeded demo and verifier accounts (`@albitor-demo.invalid`,
  `@albitor-verify.invalid`) always pass the rule.
  That exemption exists only so the platform's journeys and demo sign-in keep working. It does not
  make a restriction safe and is not a feature for app authors.
  **Do not set `SIGNUP_DOMAINS`** (in `albitor.yaml` `config:` or anywhere) unless the brief or a
  person explicitly asks for a sign-in domain restriction; never invent one.
- **App data on deletion.** Set `shell.OnAccountDeleted` from an app-owned file to remove a person's
  records after their account is deleted. It must be idempotent: the person cannot retry.
- **Public pages.** `PUBLIC_PATHS` in `guard.ts` is managed. A page that must be public (a landing page,
  say) means ejecting `guard.ts` and adding the path: say why in the delivery record. The guard only
  shapes what the browser shows; the API enforces access. An app whose every page is private sends a
  signed-out visitor straight to Cognito.

## Rules

- **Do not weaken the pool.** The pool is the deploy's, from `services.auth`: no Lambda trigger, no
  auto-confirm. The floor gates refuse all of them.
- **Do not hard-code a scope or a URL.** They come from `AUTH_LOGIN_URL` and `AUTH_LOGOUT_URL`.
- **No user-pool admin action on the API's role.** Account deletion uses the person's own access token.
- **Fail closed.** With the `AUTH_*` variables missing or malformed on anything that is not a local run,
  every `/api/` route answers 503. Do not add a fallback.
- **Do not redirect from the callback.** It answers a 200 page that navigates, because the Strict session
  cookie is not sent on a redirect that follows a cross-site navigation.

## Local run

`ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1 go run .` starts api-go's local issuer. `IdentityVerify` then
trusts it and nothing else (never beside the `AUTH_*` variables or on a deployed function), and the
hosted routes say they are not available. A spec signs in as `alice`, `bob` or `admin` through
`web/tests/helpers/local-auth.ts` (`signInAs`). With `ALBITOR_LOCAL=1` and no issuer, every request acts
for `identity.AnonymousOwner`.

## Verification

`web/tests/journeys/sign-in.spec.ts` (seeded, yours to extend) proves a signed-out visitor is handed to
the API's sign-in with where they were headed, a failed sign-in does not loop, the refusal and account
screens render and pass axe, and sign-out goes through Cognito's logout. It never types into Cognito:
the suite's `auth.setup.ts` signs in once. The universal `api-auth` floor still applies: declared
non-public endpoints return `401` unauthenticated.
