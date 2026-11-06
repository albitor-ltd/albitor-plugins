# Amazon Cognito auth plugin (`auth-cognito`)

The `auth@1` plugin for the default profile (albitor-ltd/albitor#3170, ADR-0090 sections 3 and 13),
split from the `auth` pack. A schema-2 plugin: `.claude-plugin/plugin.json` declares what it
provides, requires and contributes, and the platform composer renders it into a new app.

## What it ships

The identity core is seeded once into the app (the app owns it from then on). The sign-in calls,
screens and People admin are worked examples under `skills/auth-cognito/examples/`, laid out by
repository path, which the skill tells a coder to copy and alter.

| Repository path in the app | Seeded | What |
| --- | --- | --- |
| `api/internal/identity/` | yes | `Identity`, `Owner(r)`, `FromContext`: the one identity API handlers use |
| `api/internal/shell/identity_verify.go` | yes | `IdentityVerify` (`api.middleware`), `IdentityRoutes` (`api.routes`): `GET /api/me` |
| `api/internal/shell/identity_token.go` | yes | RS256 check against the pool's JWKS, standard library only |
| `api/internal/shell/identity_cognito.go` | yes | Pool from the environment, the session cookies, refresh |
| `api/internal/shell/identity_test.go`, `identity_hardening_test.go` | yes | The gate and the token hardening tests |
| `web/src/lib/auth/session.ts`, `guard.ts` and tests | yes | `getSession()` and the `requireSession` route guard |
| `web/src/lib/shell/scope.d/auth-cognito.ts` | yes | Exports the name `web.route-guard` resolves to |
| `web/tests/auth/provider.ts` | yes | The `auth@2` provider |
| `api/internal/shell/identity_flows.go` | example | `POST /api/auth/*`: sign-in, sign-up, confirm, reset, new-password, sign-out |
| `api/internal/shell/identity_people.go`, `identity_sigv4.go` | example | `GET /api/auth/config`, `POST /api/auth/change-password`, `POST /api/auth/delete` (self-service account deletion, `OnAccountDeleted`), the invite-only People admin, and a standard-library SigV4 signer for it |
| `api/openapi.yaml` | example | api-go's contract plus `GET /api/me` and `/api/auth/*` |
| `web/src/lib/auth/client.ts`, a self-service `guard.ts` | example | The sign-in client and a guard that redirects the self-service screens in an invite-only app |
| `web/src/routes/(auth)/` | example | Sign-in, sign-up, confirm, reset, change-password, delete-account and People screens, written against the `component-kit@2` exports (`SignInPage`, `Field`, `Button`), so they work on any kit that provides it |
| `web/src/routes/+layout.ts`, `web/src/lib/shell/AccountBar.svelte` | example | Hides the nav from a signed-out visitor; the Sign out POST form button (replaces the web-svelte shell's empty AccountBar) |
| `web/tests/journeys/sign-in.spec.ts` | example | The sign-in journey: redirect, screens, local identity |

It requires `api-runtime@1`, `web-framework@1` and `component-kit@2` (an exact major: a kit that provides only `component-kit@1` does not satisfy it), so an API-less profile uses `auth-none`. It
contributes `services.auth` (`provider: cognito`, `posture: private`) and no Terraform. It does not
set `sign_up`: the plan supplies it (default `invite_only`). The deploy target realises the pool and
hands the function `AUTH_USER_POOL_ID`, `AUTH_CLIENT_ID`, `AWS_REGION`, `AUTH_SIGN_UP`,
`AUTH_ADMIN_GROUP` and `AUTH_USER_ADMIN_GROUP`. The module stays stdlib-only, because `api/go.mod` is `api-go`'s
(ADR-0090 amendment 3).

Switching auth after the first compose keeps the files you were seeded with (they are app-owned):
remove the `/api/me` entry from `api/openapi.yaml` yourself if the app no longer serves it.

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

## Two modes: invite-only and open

`AUTH_SIGN_UP` is `services.auth.sign_up`. Unset means `open`, so an app deployed before the variable
existed behaves as it always did. `GET /api/auth/config` (public) answers `{"sign_up": ...}`, which
the screens read.

**Optional sign-up domain allow-list.** `AUTH_SIGNUP_DOMAINS` is `services.auth.signup_domains`:
`example.org` (exactly that domain) or `*.example.org` (any subdomain), comma-separated. Unset or empty
means anyone may register, as before; it is off unless the brief or the owner asks for it. When set,
`POST /api/auth/sign-up` for another domain answers 403 `email_not_allowed` with a message the screen
shows as it is, and never reaches the pool; `GET /api/auth/config` adds `signup_domains` in `open`
mode so a screen can name the domains. It judges registering only: existing accounts, invitations and
the People screen are untouched. A malformed entry closes the API rather than open it. The pool has no
Lambda trigger, so a caller who holds the public client id and talks to Cognito directly is not stopped
by it: this is a courtesy limit on the app's own sign-up, not an access control.

- **`invite_only`**: sign-up, confirm, resend, forgot and reset answer 404 `not_found`; the sign-in
  screen hides those links and `/sign-up`, `/confirm`, `/reset` redirect to `/sign-in`. User administrators
  (members of `AUTH_USER_ADMIN_GROUP`, not merely of `AUTH_ADMIN_GROUP`) manage people on `/people`: `GET`/`POST /api/auth/users`,
  `POST /api/auth/users/{disable,enable,reset-password}`. Creating a person calls `AdminCreateUser`
  with `MessageAction=SUPPRESS` and `email_verified=true` and a generated temporary password; nothing
  is emailed. The password is in the response once, with `Cache-Control: no-store`, and is never
  logged. These calls are signed with the function role's credentials, so the deploy target must
  grant that role the single-pool admin policy (`AdminCreateUser`, `AdminSetUserPassword`,
  `AdminDisableUser`, `AdminEnableUser`, `ListUsers`, `AdminAddUserToGroup`,
  `AdminListGroupsForUser`, `AdminGetUser` on the one pool), which `deploy-byoa-aws` does only in this
  mode. Without it the People calls answer 502 `unavailable`.
- **`open`**: self sign-up as before; the People endpoints answer 404.
- **Both**: `POST /api/auth/change-password` (signed in; `{current_password, new_password}`) and the
  `/change-password` screen.
- **Both**: `POST /api/auth/delete` (signed in; `{password}`) deletes the person's own account with Cognito's
  `DeleteUser`, after re-checking the password, and the `/delete-account` screen (linked from change-password).
  Every app has it, because a single-role app has no administrator to close an account for someone
  (ADR-0098). The same credentials stop signing in; `OnAccountDeleted` removes the app's own data. In an
  invite-only app the last enabled user administrator gets 409 `admin_account` until someone else is one
  (checked after the password). The session's ID token stays valid until it expires (at most an hour):
  deletion ends sign-in, not tokens already issued.

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
- Logic ships as files seeded once into the app (and examples copied from the skill), not a
  versioned module. An app owns its copy from the first compose; a fix in a later release reaches
  new apps only.

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
