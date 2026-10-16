# Amazon Cognito auth plugin (`auth-cognito`)

The `auth@1` plugin for the default profile (albitor-ltd/albitor#3170, ADR-0090 sections 3 and 13),
split from the `auth` pack. A schema-2 plugin: `.claude-plugin/plugin.json` declares what it
provides, requires and contributes, and the platform composer renders it into a new app.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `api/internal/identity/` | managed | `Identity`, `Owner(r)`, `FromContext`: the one identity API handlers use |
| `api/internal/shell/identity_verify.go` | managed | `IdentityVerify` (`api.middleware`), `IdentityRoutes` (`api.routes`): `GET /api/me`, `POST /api/auth/*` |
| `api/internal/shell/identity_token.go` | managed | RS256 check against the pool's JWKS, standard library only |
| `api/internal/shell/identity_cognito.go` | managed | Pool from the environment, the sign-in calls, the session cookies |
| `web/src/lib/auth/` | managed | `getSession()`, the `requireSession` route guard, the sign-in client, and their tests |
| `web/src/lib/shell/scope.d/auth-cognito.ts` | managed | Exports the name `web.route-guard` resolves to |
| `web/src/routes/(auth)/` | example, per kit | Sign-in, sign-up, confirm, reset and sign-out screens, written against `kit-shadcn` |
| `web/tests/journeys/sign-in.spec.ts` | example | The sign-in journey: redirect, screens, local identity |

It requires `api-runtime@1` and `web-framework@1`, so an API-less profile uses `auth-none`. It
contributes `services.auth` (`provider: cognito`, `posture: private`, `sign_up: open`) and no
Terraform: the deploy target realises the pool and hands the function `AUTH_USER_POOL_ID`,
`AUTH_CLIENT_ID` and `AWS_REGION`. The module stays stdlib-only, because `api/go.mod` is `api-go`'s
(ADR-0090 amendment 3).

## Behaviour worth knowing

- Every `/api/` request needs a verified token except `/api/health`, `/api/auth/*` and a path the
  API does not have (which stays the shell's JSON 404). Anything else is 401 `unauthenticated`.
- The browser never calls Cognito and holds no token: the API makes the public Cognito calls and
  keeps the session in HttpOnly, Secure, SameSite=Strict cookies. An expired session is refreshed
  from the refresh cookie; a bearer header works for scripts.
- Where identity comes from is decided once at start: a named pool, else the local issuer
  (`ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1`, from `api-go`, never beside a pool or on a deployed
  function), else nobody (`ALBITOR_LOCAL=1` alone: every request acts for `identity.AnonymousOwner`),
  else every route but health answers 503. It fails closed.
- The route guard reads `getSession()`, which asks `GET /api/me`: a 401 is "nobody", anything else
  is thrown rather than read as signed out. The guard never depends on a provider (the shell runs it
  before any provider mounts), and ships no provider because nothing needs one.
- The screens carry the test ids the `test-playwright` sign-in setup targets (`sign-in-form`,
  `sign-in-email`, `sign-in-password`, `sign-in-button`).

- The pool's keys (JWKS) are fetched outside any lock the other requests wait on, an unknown `kid`
  refetches at most every 5 minutes, and a failed fetch is remembered for 10 seconds. A pool that
  cannot be reached is 503, never 401, and does not end a session. A token under a `kid` the pool
  has not published yet is refused (401) but its cookies are kept, so a key rotation does not sign
  anyone out.

## Invited people: the first sign-in

A pool with `sign_up: invite_only` has people added by its owner (the deploy target's invite). Cognito
emails them a temporary password and holds the account in `FORCE_CHANGE_PASSWORD`. Signing in with it
answers `409 new_password_required` carrying the challenge `session`; the sign-in screen then asks
for a password of their own and posts `{email, password, session}` to `POST /api/auth/new-password`,
which answers the challenge (`RespondToAuthChallenge`) and sets the same session cookies as any
sign-in. The session lives in the page's memory for the few minutes Cognito honours it, never in a
URL or storage; a lapsed one answers `401 session_expired` and the person signs in again. A
repeated challenge answers `challenge_unsupported` rather than looping the screen. The accounts a
deploy seeds are permanent from the start and never meet this step.

## Requirements on the deploy target

- **The pool must be realised from `services.auth` and its environment delivered.** A deployed app
  answers 503 on every route but `/api/health` until a deploy plugin supplies `AUTH_USER_POOL_ID`
  and `AUTH_CLIENT_ID` (with `AWS_REGION`). That is deliberate, and it is how the app looks until a
  `deploy-target` plugin fills it.
- **The public client must have `PreventUserExistenceErrors` enabled.** Without it Cognito answers
  `UsernameExistsException` and `UserNotConfirmedException`, which this plugin maps to 409 `exists`
  and 403 `not_confirmed`, and an attacker can tell which email addresses have accounts. The plugin
  cannot set it (it ships no Terraform), so the deploy target's contract has to.

## Known gaps

- The screens (`web/src/routes/(auth)/`) are written for `kit-shadcn` only. With another kit the
  plugin still ships the session, the guard and the API, so the guard sends a signed-out person to
  `/sign-in` and that screen has to be written by the build.
- Sign-up and sign-in leak account existence until `PreventUserExistenceErrors` is set (above).
- No MFA, no social sign-in. The one challenge handled is `NEW_PASSWORD_REQUIRED` (below); any other
  answers 409 `challenge_unsupported`.
- Logic ships as managed files, not a versioned module (ADR-0090 amendment 3), so a fix reaches an
  app on a compose or upgrade, not silently.

## Not carried over from the `auth` pack

The Terraform (`auth.tf`), the SSM auth-config parameter, the static `index.html` screens and
`serve.mjs`, and the seeding script. Deployment is the deploy target's, the stack is Svelte, and
the demo account is seeded by the deploy identity (`deploy-byoa-aws`'s deploy step; the hosted pipeline on the hosted lane). The `auth` pack stays as released
(`auth/v0.7.0`) for the engine that still uses it.

## Verification

`scripts/verify-auth.sh auth-cognito` (run by CI) lays this plugin over `api-go`'s tree, renders the
router with this plugin's slots, and runs `gofmt`, `go build`, `go vet` and `go test`: the gate
against a fake Cognito, and sign-in through api-go's real local issuer. `tests/compose` runs the
real manifest parser, the `auth@1` conformance suite and the composer over this plugin in the
default profile. `scripts/compose-check.sh --build --journeys` builds the composed web tree and
runs the journeys in Chromium against the locally booted default shell.
