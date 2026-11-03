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

Both modes have **Change password** (`POST /api/auth/change-password`, any signed-in person) and
the public `GET /api/auth/config` (`{"sign_up": "open" | "invite_only"}`, from the `AUTH_SIGN_UP`
environment variable; unset means `open`), which the screens read to know which mode they are in.
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
| `api/internal/identity/identity.go` | `identity.Owner(r)`, `identity.FromContext(ctx)`, `Identity{Sub, Email, Groups}` and `InGroup`. The one identity API a handler uses |
| `api/internal/shell/identity_verify.go` | `IdentityVerify` (the `api.middleware` entry) refuses every `/api/` request without a verified token except `/api/health`, `/api/auth/*` and an unknown path; `IdentityRoutes` serves `GET /api/me` |
| `api/internal/shell/identity_token.go` | The RS256 check against the pool's JWKS: signature, `iss`, `token_use`, client, expiry. Standard library only |
| `api/internal/shell/identity_cognito.go` | Where the pool comes from (`signInFromEnv`), the session cookies, refresh, and the one call helper to Cognito |
| `api/internal/shell/identity_test.go`, `identity_hardening_test.go` | The gate, the token checks, key rotation, refresh and the local issuer. Each fails when the check it is named for is removed |
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
| `examples/api/internal/shell/identity_people.go`, `identity_sigv4.go` | `GET /api/auth/config`, `POST /api/auth/change-password`, the invite-only People admin, and a standard-library SigV4 signer for it. Temporary passwords are generated here, returned once with `Cache-Control: no-store`, never logged |
| `examples/api/internal/shell/identity_flows_test.go`, `identity_people_test.go`, `identity_sigv4_test.go` | Their tests, run against the seeded helpers |
| `examples/api/openapi.yaml` | The contract with `GET /api/me` and `/api/auth/*` added to the base. Merge those paths into the app's `api/openapi.yaml` |
| `examples/web/src/lib/auth/client.ts`, `client.test.ts` | `signIn`, `signUp`, `confirm`, `resend`, `forgot`, `reset`, `signOut`, `getSignUpMode`, `changePassword` and the People calls. They return `{ ok: false, code, message }` rather than throw |
| `examples/web/src/lib/auth/guard.ts`, `guard.test.ts` | A guard that also sends a visitor from `/sign-up`, `/confirm` and `/reset` to `/sign-in` in an invite-only app. Replaces the seeded guard when you adopt the self-service screens |
| `examples/web/src/routes/+layout.ts`, `examples/web/src/lib/shell/AccountBar.svelte` | The root layout load that returns `showNav: false` to a signed-out visitor (so the shell hides the app's navigation), and the shell's `AccountBar` filled in with the Sign out POST form button. Copy both over the shell's own |
| `examples/web/src/routes/(auth)/**` | Sign-in, sign-up, confirm, reset, change-password and People screens, written against the `component-kit@2` exports (`SignInPage`, `Field`, `Button`), so they work on every kit that provides it |
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

- **Authorisation.** Take the owner from `identity.Owner(r)` and put it in the partition key. Never
  take the owner from a body or a query parameter. An app has one kind of signed-in account unless
  the plan declares roles: never invent roles or an admin group. A route that only some people may
  use gates on `id.InGroup(<the declared admin_group>)` after `identity.FromContext`, and that group
  must be listed in `services.auth.groups`. The platform seeds the demo and verifier accounts into
  `admin_group` on every dev deploy; never ask a person to add accounts to groups or change roles,
  that is the platform's job.
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
  page, say) means adding the path there: say why in the delivery record. The
  guard only shapes what the browser shows; the API enforces access.
- **Anything the brief adds** (MFA, social sign-in) is a change to `services.auth` in
  `albitor.yaml` and, where the API must change, an edit to the file that needs it. `sign_up` is `open` or
  `invite_only` (the plan sets it; do not hard-code it in the app); a challenge the screens do not handle answers `challenge_unsupported`. The
  invited person's temporary-password step (`new_password_required`, then `newPassword` in
  `client.ts`) is handled: keep the `new-password-*` test ids on the sign-in page.

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
