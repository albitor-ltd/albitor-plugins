---
name: auth-cognito
description: Use when building or reviewing sign-in for an app composed with the auth-cognito plugin (auth@1) - who owns a record, which routes need a person, a group-gated action, the sign-in, sign-up, confirm or reset screens, the People admin, the route guard, or signing in as a test identity in a local run. The identity seam, the session module and the guard are in the repository; the sign-in calls, screens and People admin are worked examples to copy, then alter.
metadata:
  capability: auth
---

# Cognito auth (auth-cognito)

A person can sign in and reach a protected route. How they get an account is
`services.auth.sign_up`, which the plan supplies (the plugin does not set it) and which defaults to
`invite_only`:

- **`invite_only`** (the default). Nobody registers themselves. A user administrator (a member of the
  dedicated `AUTH_USER_ADMIN_GROUP`, which holds the owner and whoever the owner promotes; the
  platform's demo and verifier accounts are in the app's admin group but not in it) adds people on the **People** screen: the API creates the account with a generated
  temporary password, sends no email, and shows the password once for the administrator to pass on.
  The person signs in with it and is asked for a password of their own (`new_password_required`).
  Sign-up, confirm, resend, forgot and reset answer 404 `not_found`; the sign-in screen hides those
  links and `/sign-up`, `/confirm` and `/reset` send a visitor to `/sign-in`. An administrator can
  also disable, enable and reset the password of a person. People endpoints are live only here, and with no
  `AUTH_USER_ADMIN_GROUP` nobody can use them (403). "Make an administrator" adds the person to both
  `AUTH_ADMIN_GROUP` and the user-admin group.
- **`open`** (self sign-up). A first-time visitor registers, confirms their email, signs in. The
  People endpoints answer 404 and nothing in the pool changes from before.
  **An unconfirmed person is never stuck, and never deleted.** Someone who signed up but never
  entered the emailed code can always get a new one: sign-in answering `not_confirmed` and sign-up
  answering `exists` both lead to `/confirm?email=<email>&resend=1`, which sends a new code once on
  arrival (and drops the parameter so a reload does not resend). With `PreventUserExistenceErrors`
  on (required), a confirmed account looks the same as an unconfirmed one: resend is a fake success
  and `confirm` answers `invalid_code`. So the screen never claims a send for certain ("If an account
  with <email> is waiting for confirmation, we've sent it a new code") and always shows "Already
  confirmed? Sign in". `resend` and `confirm` answer 409 `already_confirmed` only when the flag is
  off. `expired_code` tells the person to send a new code, and a rate limit (429 `rate_limited`)
  leaves the button usable. Keep this when you alter the screens.

Both modes have **Change password** (`POST /api/auth/change-password`, any signed-in person) and
the public `GET /api/auth/config` (`{"sign_up": "open" | "invite_only"}`, from the `AUTH_SIGN_UP`
environment variable; unset means `open`), which the screens read to know which mode they are in.
`services.auth.signup_domains` (optional, off by default: never set it unless the brief or the owner
names the domains) reaches the function as `AUTH_SIGNUP_DOMAINS`; `sign-up` for another domain answers
403 `email_not_allowed` and `GET /api/auth/config` adds `signup_domains` in `open` mode. The list is
held on every token (`holdsSession` in the seeded identity core), so it also refuses existing users and
invitees on unlisted domains, and an address the pool has not verified (`email_verified` must be true):
it is chosen before launch, or it includes every current user's domain. The platform's own accounts
pass by membership of the `platform-seeded` pool group, not by address.
Keep the sign-up and sign-in screens showing the API's message.
All of that is in the repository: read it, then extend it. The plugin declares `services.auth`
(provider `cognito`, posture `private`) in `albitor.yaml`; the deploy target realises the user pool
and hands the function `AUTH_USER_POOL_ID`, `AUTH_CLIENT_ID`, `AWS_REGION`, `AUTH_SIGN_UP`,
`AUTH_ADMIN_GROUP` and `AUTH_USER_ADMIN_GROUP`. There is no Terraform and no pool id in the repository.

## Seeded files (the app's own)

These are written once and are plain app code. Change them when the app needs it, and keep the
identity API (`identity.Owner`, `identity.FromContext`) and the checks the API makes on every token,
which is what keeps one person's records from another.

| File | What it does |
| --- | --- |
| `api/internal/identity/identity.go` | `identity.Owner(r)` (returns `(owner, error)`), `identity.FromContext(ctx)`, `Identity{Sub, Email, Groups}` and `InGroup`. The one identity API a handler uses |
| `api/internal/shell/identity_public.go`, `identity_public_test.go` | The browse posture: `AUTH_POSTURE` (unset is private), `AUTH_PUBLIC_ROUTES` and `AUTH_PUBLIC_BODY_MAX` parsed (a malformed one fails closed), the route matcher and the public-write body cap. The gate lets a listed route through with optional identity (below) |
| `api/internal/shell/identity_verify.go` | `IdentityVerify` (the `api.middleware` entry) refuses every request, on any path, without a verified token, except `/api/health`, the exact sign-in calls in `publicRoutes` (`POST /api/auth/sign-in`, `sign-up`, `confirm`, `resend`, `forgot`, `reset`, `new-password`, `sign-out` and `GET /api/auth/config`) and a path nothing serves (the JSON 404); nothing matches by prefix, so a route you add under `/api/auth/` or outside `/api/` is protected unless you name it in `publicRoutes`; `IdentityRoutes` serves `GET /api/me` |
| `api/internal/shell/identity_token.go` | The RS256 check against the pool's JWKS: signature, `iss`, `token_use`, client, expiry. Standard library only |
| `api/internal/shell/identity_cognito.go` | Where the pool comes from (`signInFromEnv`), the session cookies, refresh, and the one call helper to Cognito |
| `api/internal/shell/identity_test.go`, `identity_hardening_test.go` | The gate, the token checks, key rotation, refresh and the local issuer. Each fails when the check it is named for is removed |
| `api/internal/identity/owned_data_scope_test.go` | `TestOwnedDataScope`: the two-owner contract test over the app's per-user routes (see the api-go skill). Keep it and register a factory per per-user entity; never weaken it |
| `web/src/lib/auth/session.ts` | `getSession()`, which asks `GET /api/me`. Provider-independent: the guard and the screens both read it |
| `web/src/lib/auth/guard.ts` | `requireSession`, the `web.route-guard` entry. Sends a signed-out person to `/sign-in?next=...` |
| `web/src/lib/shell/scope.d/auth-cognito.ts` | Exports the name the `web.route-guard` slot resolves to |
| `web/tests/auth/provider.ts` | The `auth@2` provider the specs call |

Out of the box a person cannot yet sign in: the sign-in calls and the screens are the examples
below. The seeded API answers `GET /api/me` and refuses everything else without a token.

## Examples (copy, then alter)

Worked examples sit in `examples/` beside this skill, laid out by repository path: copy a file to the
same path in the app. They are the app's code once copied; they never reference this skill. The
foundation node writes the sign-in shell from them.

| Example | What it is |
| --- | --- |
| `examples/api/internal/shell/identity_flows.go` | `registerAuthRoutes` and the sign-in calls: `POST /api/auth/sign-in`, `sign-up`, `confirm`, `resend`, `forgot`, `reset`, `new-password`, `sign-out`, and the Cognito error mapping. `sign-out` also takes a same-site form post (SvelteKit's form-action CSRF rule: `Sec-Fetch-Site` same-origin, or `Origin` equal to the host) and answers `303` to `/sign-in` |
| `examples/api/internal/shell/identity_people.go`, `identity_sigv4.go` | `GET /api/auth/config`, `POST /api/auth/change-password`, `POST /api/auth/delete` (self-service account deletion with Cognito's `DeleteUser`, and the `OnAccountDeleted` hook), the invite-only People admin, and a standard-library SigV4 signer for it. Temporary passwords are generated here, returned once with `Cache-Control: no-store`, never logged |
| `examples/api/internal/shell/identity_flows_test.go`, `identity_people_test.go`, `identity_sigv4_test.go` | Their tests, run against the seeded helpers |
| `examples/api/internal/shell/owned_data_scope_test.go` | `TestOwnedDataScopePeopleAdminRoutes`: the two-owner contract test for the People admin routes, which the contract declares admin-only (`x-admin-only: true`); a non-admin must be answered 403. Copy it with the People routes; it sets `Only: /api/auth/` so the app's own per-user entities are left to its own test. The seeded `internal/identity` test leaves `/api/auth/` to it (`Elsewhere`) |
| `examples/api/openapi.yaml` | The contract with `GET /api/me` and `/api/auth/*` added to the base, the People routes marked `x-admin-only: true`. Merge those paths into the app's `api/openapi.yaml` |
| `examples/web/src/lib/auth/client.ts`, `client.test.ts` | `signIn`, `signUp`, `confirm`, `resend`, `forgot`, `reset`, `signOut`, `getSignUpMode`, `changePassword`, `deleteAccount` and the People calls. They return `{ ok: false, code, message }` rather than throw |
| `examples/web/src/lib/auth/guard.ts`, `guard.test.ts` | A guard that also sends a visitor from `/sign-up`, `/confirm` and `/reset` to `/sign-in` in an invite-only app. Replaces the seeded guard when you adopt the self-service screens |
| `examples/web/src/routes/+layout.ts`, `examples/web/src/lib/shell/AccountBar.svelte` | The root layout load that returns `showNav: false` to a signed-out visitor, except on a declared public page (browse posture), plus `signedIn`; and the shell's `AccountBar`, which holds the Sign out POST form button signed in and Sign in / Sign up links signed out. Copy both over the shell's own |
| `examples/web/src/routes/(auth)/**` | Sign-in, sign-up, confirm, reset, change-password, delete-account and People screens, written against the `component-kit@2` exports (`SignInPage`, `Field`, `Button`), so they work on every kit that provides it |
| `examples/web/tests/journeys/sign-in.spec.ts` | The sign-in journey: redirect, screens, local identity |

To adopt the sign-in calls, copy the three API files together (`identity_flows.go` hands the
signed-in routes to `identity_people.go`, which signs with `identity_sigv4.go`), then add one line at
the top of `registerIdentityRoutes` in `identity_verify.go`:

```go
registerAuthRoutes(mux, setup, logger)
```

The routes `/sign-in`, `/sign-up`, `/confirm`, `/reset` and `/sign-out` are fixed (there is no `/sign-out` page: signing out is a POST, below): seeded `guard.ts` hard-code them, so a screen keeps its path.

The browser never talks to Cognito (the page's CSP allows its own origin only) and no token is
readable by page script. A script or test client may send `Authorization: Bearer <token>` instead.

## What the build writes

- **Authorisation.** Take the owner from `owner, err := identity.Owner(r)` (answer 401 on `err`: a deployed function with no identity never acts as an anonymous owner) and put it in the partition key. Never
  take the owner from a body or a query parameter. An app has one kind of signed-in account unless
  the plan declares roles: never invent roles or an admin group. A route that only some people may
  use gates on `id.InGroup(<the declared admin_group>)` after `identity.FromContext`, and that group
  must be listed in `services.auth.groups`. The platform seeds the demo and verifier accounts into
  `admin_group` on every dev deploy; never ask a person to add accounts to groups or change roles,
  that is the platform's job.
- **Account deletion is the person's own, in every app.** Keep `POST /api/auth/delete` and the
  `/delete-account` screen (linked from change-password): it deletes the account in the pool with
  `DeleteUser`, so the same email and password no longer sign in. A single-role app has no
  administrator and no People screen (ADR-0098), so never write a deletion dialog that says an
  administrator "closes the sign-in", and never ship a Delete button that only hides the account or
  returns 204 while the credentials still work. The data the app holds for the person is removed by
  `shell.OnAccountDeleted`, which the app sets from its own code (an `init` in a file of its own); make
  it idempotent. In an invite-only app the last enabled user administrator is refused (409
  `admin_account`). The ID token already issued stays valid until it expires (at most an hour). The API's test `TestDeleteAccountRemovesTheAccountSoItCannotSignInAgain` proves the
  credentials stop working: keep it. A journey spec must not delete a real account (the seeded one is
  the suite's only way in), so the example journey answers the deletion call itself.
- **Roles in specs (`auth@2`).** `web/tests/auth/provider.ts` exports `auth` with `roles()`
  (the declared `services.auth.groups`), `seed(role, { request })` and `signInAs(role, { context,
  request, baseURL })`. A spec that needs to be a role calls these and nothing else: never a
  hand-made user, never a password in a spec. Each role's account holds that group and no other. Locally it is the `api-go` local issuer; deployed it is the seeded `<role>@albitor-role.invalid` signed in on the app's own form.
- **The screens** in `web/src/routes/(auth)/` (copied from the examples) are yours: restyle and reword them,
  add sign-up fields (and the field in `client.ts`'s call), but **keep the `data-testid` values**
  on the sign-up, confirm, reset and new-password forms. The sign-in form is the kit's `SignInPage`
  (component-kit@2), found by role and label; keep it rather than rebuilding it. **Sign out is a POST form button, never a link** (`AccountBar.svelte`; it awaits the hooks the app registers with `onBeforeSignOut` from `lib/shell/signout.ts`, which is where the data layer's local wipe goes: see its skill; auth never imports a data layer): a link can be prefetched
  and would sign people out. Never add a GET route or nav entry that signs out. The app's navigation
  is shown only when `getSession()` finds someone (`+layout.ts`'s `showNav`). `change-password/` has its own `nav.ts`. The People screen has none (the nav is
  static); the change-password screen links to it for user-admin-group members, which
  `GET /api/auth/config` names as `user_admin_group`. People keeps its `people-*` test ids, shows the
  one-time password with a copy button, and nudges the administrator to add a second one while
  there is only one user administrator.
- **Public pages.** `PUBLIC_PATHS` in `guard.ts` lists the public paths (the sign-in, sign-up, confirm,
  reset, sign-out and offline paths are in it already). A page that must be public (a landing
  page, say) means adding the path there and, in posture public, listing it in
  `services.auth.public_routes` (see Browse posture below): say why in the delivery record. The
  guard only shapes what the browser shows; the API enforces access.
- **Anything the brief adds** (MFA, social sign-in) is a change to `services.auth` in
  `albitor.yaml` and, where the API must change, an edit to the file that needs it. `sign_up` is `open` or
  `invite_only` (the plan sets it; do not hard-code it in the app); a challenge the screens do not handle answers `challenge_unsupported`. The
  invited person's temporary-password step (`new_password_required`, then `newPassword` in
  `client.ts`) is handled: keep the `new-password-*` test ids on the sign-in page.

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

- **Account specs in a browse app fail, never skip.** With `posture: public` an account or role journey
  guards on `requireSession(test, why)` (`web/tests/helpers/storage-state.ts`), which fails the spec when the
  suite has no session; do not guard it on `hasDemoSession()`, whose skip reads green. A public journey guards
  on nothing: it starts with `test.use({ storageState: EMPTY_STORAGE_STATE })` and runs signed out.


## Rules

- **Do not weaken the pool.** The pool is the deploy target's, from `services.auth`: no Lambda
  trigger, no auto-confirm, at least 12 characters. The floor gates refuse all of them.
- **No user-pool admin action on the API's role, except the module-owned one.** Seed demo accounts
  with the deploy identity. The only sanctioned admin access is the deploy target's own single-pool
  policy (create, enable, disable, set-password, list and group lookups on this one pool), which it
  attaches only when `sign_up` is `invite_only` and self sign-up is off. Never write an IAM grant,
  a `cognito-idp:Admin*` statement or a `user_admin_enabled` flag into the app's own tree; in an
  `open` app there is no admin access at all and the People endpoints answer 404.
- **Never add an auto-confirm bypass, in either mode.** No `pre_sign_up` trigger, no
  `lambda_config`, no auto-confirm flag. The temporary password is the invitation: it is never
  logged, stored or emailed by the app.
- **Fail closed.** With no pool configured on anything that is not a local run, every `/api/`
  route answers 503. Do not add a fallback.
- **Only one spec registers** (an `open` app). A sign-up sends an email, and the default Cognito sender allows 50 a
  day per AWS account. Other specs use the suite's seeded session or a local identity.
- **No Cognito Hosted UI.** The screens are the app's own.

## Local run

`ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1 go run .` starts api-go's local issuer. `IdentityVerify`
then trusts it and nothing else (never beside a named pool or on a deployed function), and the
sign-in calls answer 501 `local`. A spec signs in as `alice`, `bob` or `admin` through
`web/tests/helpers/local-auth.ts` (`signInAs`); the journey harness does this for every journey
test. With `ALBITOR_LOCAL=1` and no issuer, every request acts for `identity.AnonymousOwner`.

## Verification

`web/tests/journeys/sign-in.spec.ts` (copied from the examples, yours to extend) proves a signed-out visitor is sent
to sign-in, the screens render and pass axe, and a local identity reaches the app. The universal
`api-auth` floor still applies: declared non-public endpoints return `401` unauthenticated.
