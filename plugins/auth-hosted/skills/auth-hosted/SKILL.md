---
name: auth-hosted
description: Use when building or reviewing sign-in for a hosted-lane app composed with the auth-hosted plugin (auth@1) - who owns a record, which routes need a person, a group-gated action, the sign-in redirect, sign-out, the account and deletion screens, the sign-up domain refusal, the route guard, or acting as a test identity in a local run. The identity seam, the session module and the guard are in the repository; the hosted sign-in routes and the screens are worked examples to copy, then alter.
metadata:
  capability: auth
---

# Hosted sign-in (auth-hosted)

A first-time visitor is sent to the app's Cognito hosted page, signs up or in there, and comes back
signed in. All of the app's side is in the repository: read it, then extend it. The plugin declares
`services.auth` (provider `cognito`, posture `private`; the plan's browse mode sets `public` and `public_routes`) in `albitor.yaml`; the plan's access pick supplies `sign_up`; the hosted
deploy realises the pool and hands the function `AUTH_ISSUER`, `AUTH_CLIENT_ID`, `AUTH_JWKS_URL`,
`AUTH_POSTURE`, `AUTH_LOGIN_URL` and `AUTH_LOGOUT_URL`. There is no Terraform and no pool id in the
repository.

## Seeded files (the app's own)

These are written once and are plain app code. Change them when the app needs it, and keep the
identity API (`identity.Owner`, `identity.FromContext`) and the checks the API makes on every token,
which is what keeps one person's records from another.

| File | What it does |
| --- | --- |
| `api/internal/identity/identity.go` | `identity.Owner(r)` (returns `(owner, error)`), `identity.FromContext(ctx)`, `Identity{Sub, Email, Groups}` and `InGroup`. The one identity API a handler uses |
| `api/internal/shell/identity_verify.go` | `IdentityVerify` (the `api.middleware` entry) refuses every request, on any path, without a verified token, except `/api/health`, the exact sign-in routes in `publicRoutes` (`GET /api/auth/login` and `callback`, `POST /api/auth/sign-out`), `POST /events` only on a schedule or queue function (the adapter's `AWS_LWA_PASS_THROUGH_PATH` is `/events`; on the public HTTP function it needs a verified identity) and a path nothing serves (the JSON 404); nothing matches by prefix, so a route you add under `/api/auth/` or outside `/api/` is protected unless you name it in `publicRoutes`; `IdentityRoutes` serves `GET /api/me` |
| `api/internal/shell/identity_public.go`, `identity_public_test.go` | The browse posture: `AUTH_POSTURE`, `AUTH_PUBLIC_ROUTES` and `AUTH_PUBLIC_BODY_MAX` parsed (a malformed one fails closed), the route matcher and the public-write body cap. The gate lets a listed route through with optional identity (below) |
| `api/internal/shell/identity_token.go` | The RS256 check against the pool's JWKS: signature, `iss`, `token_use`, client, expiry. Standard library only |
| `api/internal/shell/identity_hosted.go` | The environment, the ID-token gate and the `SIGNUP_DOMAINS` rule, refresh at the token endpoint, and the `__Host-` session cookies (HttpOnly, Secure, SameSite=Strict) |
| `api/internal/shell/identity_test.go` | The gate, the token checks, the email and domain rule on every request and refresh, key rotation and the local issuer |
| `api/internal/identity/owned_data_scope_test.go` | `TestOwnedDataScope`: the two-owner contract test over the app's per-user routes (see the api-go skill). Keep it and register a factory per per-user entity; never weaken it |
| `web/src/lib/auth/session.ts` | `getSession()`, which asks `GET /api/me`. Provider-independent: the guard and the screens both read it |
| `web/src/lib/auth/guard.ts` | `requireSession`, the `web.route-guard` entry. Sends a signed-out person to `/sign-in?next=...` |
| `web/src/lib/shell/scope.d/auth-hosted.ts` | Exports the name the `web.route-guard` slot resolves to |
| `web/tests/auth/provider.ts` | The `auth@2` provider the specs call |

Out of the box a person cannot yet sign in: the hosted sign-in routes and the screens are the
examples below. The seeded API answers `GET /api/me` and refuses everything else without a token.

## Examples (copy, then alter)

Worked examples sit in `examples/` beside this skill, laid out by repository path: copy a file to the
same path in the app. They are the app's code once copied; they never reference this skill. The
foundation node writes the sign-in shell from them.

| Example | What it is |
| --- | --- |
| `examples/api/internal/shell/identity_flows.go` | `registerAuthRoutes` and the hosted flow: `GET /api/auth/login` and `/callback` (authorization code with PKCE), `POST /api/auth/sign-out` (JSON, which the Sign out button fetches; a same-site form post answered `303` to the logout URL for a client with no script; any client answered locally in a local run) and `/delete`, and `OnAccountDeleted` |
| `examples/api/internal/shell/identity_flows_test.go` | The flow's tests: state and verifier, the callback's order of checks, cookies, sign-out, deletion, CSRF |
| `examples/api/openapi.yaml` | The contract with `GET /api/me` added to the base. Merge that path into the app's `api/openapi.yaml` |
| `examples/web/src/lib/auth/client.ts`, `client.test.ts` | `loginUrl`, `signOut`, `signOutAndLeave` (fetch, then `location`), `deleteAccount`. They return `{ ok: false, code, message }` rather than throw |
| `examples/web/src/routes/(auth)/**` | The sign-in, account, account deletion and refusal screens, on the `component-kit@2` exports |
| `examples/web/src/routes/+layout.ts`, `examples/web/src/lib/shell/AccountBar.svelte` | `+layout.ts` shows the nav only to a signed-in person and fails closed (`canShowNav`); `AccountBar` is the Sign out POST form button. Sign out is never a link. Register a data-layer wipe with `onBeforeSignOut` (`lib/shell/signout.ts`) |
| `examples/web/tests/journeys/sign-in.spec.ts` | The sign-in journey |

To adopt the hosted routes, copy `identity_flows.go` and its test, then add one line at the top of
`registerIdentityRoutes` in `identity_verify.go`:

```go
registerAuthRoutes(mux, setup, logger)
```

The routes `/sign-in` and `/not-allowed` are fixed: seeded `guard.ts` and `identity_hosted.go` hard-code them, so a screen keeps its path.

The browser never talks to Cognito's API and no token is readable by page script. A script or test
client may send `Authorization: Bearer <token>` instead.

## What the build writes

- **Authorisation.** Take the owner from `owner, err := identity.Owner(r)` (answer 401 on `err`: a deployed function with no identity never acts as an anonymous owner) and put it in the partition key. Never
  take the owner from a body or a query parameter. An app has one kind of signed-in account unless
  the plan declares roles: never invent roles or an admin group. A route that only some people may
  use gates on `id.InGroup(<the declared admin_group>)` after `identity.FromContext`, and that group
  must be listed in `services.auth.groups`. The platform seeds the demo and verifier accounts into
  `admin_group` on every dev deploy; never ask a person to add accounts to groups or change roles,
  that is the platform's job.
- **Roles in specs (`auth@2`).** `web/tests/auth/provider.ts` exports `auth` with `roles()`
  (the declared `services.auth.groups`), `seed(role, { request })` and `signInAs(role, { context,
  request, baseURL })`. A spec that needs to be a role calls these and nothing else: never a
  hand-made user, never a password in a spec. Each role's account holds that group and no other. Locally it is the `api-go` local issuer; deployed it is the seeded `<role>@albitor-role.invalid` signed in on the Cognito hosted page.
- **The screens** in `web/src/routes/(auth)/` (copied from the examples) are yours: restyle and reword them. Keep
  the `data-testid` values and the headings the journey spec reads. There is **no password field**: do
  not add a sign-in form, a sign-up screen or a reset screen, because Cognito's pages do those.
  Sign-up mode (`open` by default, `invite_only` optional) is the plan's access pick, not this plugin's: do not add the `auth-cognito`
  invite-only People screen, change-password route or `AUTH_SIGN_UP` handling here.
- **Who may sign in.** `SIGNUP_DOMAINS` (comma-separated `example.org` or `*.example.org`) limits sign-in
  by email domain. A refused person lands on `/not-allowed`; reword that page and say who to contact. The rule and `email_verified` are applied to every ID token the API takes (callback, refresh, every request), so a token from a hand-run flow gets nowhere; the gate accepts ID tokens only.
  The platform's seeded demo and verifier accounts (`@albitor-demo.invalid`,
  `@albitor-verify.invalid`) always pass the rule.
  That exemption exists only so the platform's journeys and demo sign-in keep working. It does not
  make a restriction safe and is not a feature for app authors.
  **Do not set it** unless the brief or a person explicitly asks for a sign-in domain restriction; never
  invent one. The way to set it is `services.auth.signup_domains` in `albitor.yaml` (a list of
  `example.org` / `*.example.org`), which the compiler renders as `SIGNUP_DOMAINS`; do not also declare
  the key under `config:` (that is a `duplicate_name` error). The list holds every token on both lanes,
  so it also refuses existing users and invitees on unlisted domains: set it before launch or include
  every current user's domain. Use punycode (`xn--`) for IDN domains.
- **App data on deletion.** Set `shell.OnAccountDeleted` from an app-owned file to remove a person's
  records after their account is deleted. It must be idempotent: the person cannot retry.
- **Public pages.** `PUBLIC_PATHS` in `guard.ts` lists the public paths. A page that must be public (a landing page,
  say) means adding the path there and, in posture public, listing it in `services.auth.public_routes`
  (see Browse posture below): say why in the delivery record. The guard only
  shapes what the browser shows; the API enforces access. An app whose every page is private sends a
  signed-out visitor straight to Cognito.

## Browse posture (public routes)

With `services.auth.posture: public` and a non-empty `services.auth.public_routes` the app lets
anyone browse and asks for an account only for more. The API reads `AUTH_POSTURE`,
`AUTH_PUBLIC_ROUTES` (the entries exactly as written in `albitor.yaml`, joined by commas) and
`AUTH_PUBLIC_BODY_MAX` (bytes). The platform compiles all three; the app never sets them. Nothing is
public unless listed: everything else stays default-deny on the API and in the web guard.

- **API.** A web entry (starts with `/`) is the guard's; the gate ignores it. An API entry is
  `METHOD /api/path`, where `{param}` matches one segment and a trailing `/*` matches one or more
  further segments. A public `GET` also answers `HEAD`. No token: the handler runs anonymous. A valid
  token: verified and used, so a signed-in person sees their own saved items. A token that is present
  but bad, expired or for another client answers 401, never a silent downgrade. A non-`GET` public
  route refuses a body over `AUTH_PUBLIC_BODY_MAX` with 413 (a body of unknown length errors when read
  past it). A malformed list, an unknown posture or a public posture with no list closes the whole API
  (503), like the other variables.
- **Handlers on a public route cope with no identity.** `identity.FromContext(r.Context())` answers
  `ok == false` when anonymous; `identity.Owner(r)` returns `ErrNoIdentity` in a deployed function
  and never a shared owner, so do not use it for the anonymous case. A public read returns the same
  shared data to everyone and adds the person's own items only when `FromContext` finds one:

  ```go
  func listListings(w http.ResponseWriter, r *http.Request) {
  	rows := store.Listings(r.Context())
  	if id, ok := identity.FromContext(r.Context()); ok {
  		saved = store.SavedBy(r.Context(), id.Sub) // personal extras only when signed in
  	}
  	// ...
  }
  ```

- **Public write handlers validate every field they accept.** The platform caps the body and nothing
  else: there is no rate limit and no bot check. Check types, lengths and allowed values, reject unknown
  fields, take no owner from the body, and write only what the route is for. The probe sends an empty
  body and expects a 4xx that is not 401, and an over-cap body and expects 413.
- **Per-journey roles.** When the plan gives a journey a list of roles, gate that journey's screens and
  its API routes on the groups with the app's one authorisation check, `id.InGroup(<group>)`, after
  `identity.FromContext`; answer 403 to a signed-in person without the group and 401 to nobody. Never
  invent a second check or a role the plan did not declare. A journey marked `account` needs only a
  signed-in person; a `public` journey's routes go in `public_routes`.
- **Web.** `PUBLIC_PATHS` in `web/src/lib/auth/guard.ts` is one literal array of strings: the auth pages
  first, then the manifest's web entries (`/listings`, `/listings/*`), added by the foundation node.
  The floor's manifest check reads that array and fails the tree when it disagrees with
  `public_routes`, so do not build it from spreads or imports, and add nothing to it that the manifest
  does not list. A `/*` entry matches below its prefix only (`/listings/*` is not `/listings`). On a
  declared public page the root `+layout.ts` shows the nav signed out (`showNav`) and returns
  `signedIn`; `AccountBar` then shows Sign in and Sign up, carrying the page as `next`. A private link
  opened signed out goes to `/sign-in?next=...`, and sign-in returns to `next` only when it is a
  same-origin path that is not an auth page (`nextPath`).

## Unconfirmed people

Someone who signed up but never entered the emailed code is unconfirmed. Managed login offers
"Resend code" only on the confirm screen straight after sign-up; signing in while unconfirmed
shows "User is not confirmed", and forgot-password fails. Recovery is to sign up again with the
same address: a `terraform-aws-hosted-app` release that carries PR #40 has a PreSignUp trigger
that replaces the unconfirmed user and sends a fresh code. The verification email states the code
lifetime (24 hours; 1 hour for a password reset). The app adds no sign-up or confirm screen of
its own.

## Rules

- **Do not weaken the pool.** The pool is the deploy's, from `services.auth`: the app adds no Lambda
  trigger and no auto-confirm (the module's own PreSignUp trigger is the deploy's). The floor gates
  refuse all of them.
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

`web/tests/journeys/sign-in.spec.ts` (copied from the examples, yours to extend) proves a signed-out visitor is handed to
the API's sign-in with where they were headed, a failed sign-in does not loop, the refusal and account
screens render and pass axe, and the Sign out button signs out by fetch and then navigates to Cognito's logout (never a form post answered by a redirect to Cognito: the hosted CSP's `form-action 'self'` blocks it). It never types into Cognito:
the suite's `auth.setup.ts` signs in once. The universal `api-auth` floor still applies: declared
non-public endpoints return `401` unauthenticated.
