# Amazon Cognito auth plugin (`auth-cognito`)

The `auth@1` plugin for the default profile (albitor-ltd/albitor#3170, ADR-0090 sections 3 and 13),
split from the `auth` pack. A schema-2 plugin: `.claude-plugin/plugin.json` declares what it
provides, requires and contributes, and the platform composer renders it into a new app.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `api/internal/identity/` | starting point | `Identity`, `Owner(r)`, `FromContext`: the one identity API handlers use |
| `api/internal/shell/identity_verify.go` | starting point | `IdentityVerify` (`api.middleware`), `IdentityRoutes` (`api.routes`): `GET /api/me`, `POST /api/auth/*` |
| `api/internal/shell/identity_token.go` | starting point | RS256 check against the pool's JWKS, standard library only |
| `api/internal/shell/identity_cognito.go` | starting point | Pool from the environment, the sign-in calls, the session cookies |
| `api/openapi.yaml` | starting point | api-go's contract plus the `GET /api/me` entry (200 identity, 401 signed out) |
| `web/src/lib/auth/` | starting point | `getSession()`, the `requireSession` route guard, the sign-in client, and their tests |
| `web/src/lib/shell/scope.d/auth-cognito.ts` | starting point | Exports the name `web.route-guard` resolves to |
| `web/src/routes/(auth)/` | example | Sign-in, sign-up, confirm, reset and sign-out screens, written against the `component-kit@2` exports (`SignInPage`, `Field`, `Button`), so they work on any kit that provides it |
| `web/tests/journeys/sign-in.spec.ts` | example | The sign-in journey: redirect, screens, local identity |

It requires `api-runtime@1`, `web-framework@1` and `component-kit@2` (an exact major: a kit that provides only `component-kit@1` does not satisfy it), so an API-less profile uses `auth-none`. It
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
- The sign-in form is the kit's `SignInPage`, which takes no test ids, so the journey spec finds it by
  role and label. The sign-up, confirm and reset forms and the new-password step keep their
  `data-testid` values.

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

- The page frame (`AppShell`, and its optional `footer`) is web-svelte's root layout, not this
  plugin's; the screens add no footer of their own.
- Sign-up and sign-in leak account existence until `PreventUserExistenceErrors` is set (above).
- No MFA, no social sign-in. The one challenge handled is `NEW_PASSWORD_REQUIRED` (below); any other
  answers 409 `challenge_unsupported`.
- Logic ships as files seeded once into the app, not a versioned module. An app owns its copy from
  the first compose; a fix in a later release reaches new apps only.

## Upgrading from 0.2.0

0.3.0 requires `component-kit@2`, so the kit must be `kit-shadcn`, `kit-carbon` or `kit-gds` at 0.5.0
or later (they declare both majors). The screens are app-owned examples, so an existing app keeps its 0.2.0 pages, and an app keeping
the 0.2.0 sign-in page needs nothing more: it keeps working (the `kit/index.ts` exports it uses are
unchanged) but carries the old `sign-in-*` test ids. Adopting the new page means taking the new spec:
`(auth)/sign-in/+page.svelte` and `tests/journeys/sign-in.spec.ts` together, since the spec no longer
looks for the old test ids. What compose does with example files on an upgrade is under review in
albitor-ltd/albitor#3722, so check `web/src/routes/(auth)/` after upgrading rather than relying on
either behaviour. `test-playwright` 0.2.0 finds the sign-in form by label and role ("Email...",
"Password", "Sign in"), which the 0.2.0 page's labels already satisfy.

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
