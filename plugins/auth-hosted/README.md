# Hosted sign-in auth plugin (`auth-hosted`)

The `auth@1` plugin for the hosted lane. A schema-2 plugin: `.claude-plugin/plugin.json` declares what
it provides, requires and contributes, and the platform composer renders it into a new app.

`auth-cognito` does not work on the hosted lane. The hosted-app auth module's Cognito client is public
(no secret) and allows only the OAuth authorization code flow and refresh, not `USER_PASSWORD_AUTH`,
and the module hands the function `AUTH_ISSUER`, `AUTH_CLIENT_ID`, `AUTH_JWKS_URL` and friends, not a
pool id and a region. So `auth-cognito` answers 503 there. `auth-hosted` is the provider for that
client; `auth-cognito` stays the plugin for an app's own account (BYOA), unchanged.

People sign in on the app's own Cognito hosted page (the classic hosted UI). **The app has no password
field, no sign-up screen and no reset screen**: Cognito's pages do sign-up, email confirmation,
password reset, an invited person's new password and any MFA.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `api/internal/identity/` | managed | `Identity`, `Owner(r)`, `FromContext`: the one identity API handlers use (the same as `auth-cognito`'s) |
| `api/internal/shell/identity_verify.go` | managed | `IdentityVerify` (`api.middleware`), `IdentityRoutes` (`api.routes`): `GET /api/me`, `/api/auth/*` |
| `api/internal/shell/identity_token.go` | managed | RS256 check against the pool's JWKS, standard library only |
| `api/internal/shell/identity_hosted.go` | managed | The environment, login, callback, refresh, sign-out, account deletion, the sign-up domain rule, the cookies |
| `web/src/lib/auth/` | managed | `getSession()`, the `requireSession` route guard, the API calls, and their tests |
| `web/src/lib/shell/scope.d/auth-hosted.ts` | managed | Exports the name `web.route-guard` resolves to |
| `web/src/routes/(auth)/` | example | Sign-in (redirect), sign-out, account, delete-account confirmation and refusal screens, written against the `component-kit@2` exports (`Button`) |
| `web/tests/journeys/sign-in.spec.ts` | example | The sign-in journey: the hand-over, a failed sign-in that does not loop, the refusal page, account and sign-out |

It requires `api-runtime@1`, `web-framework@1`, `component-kit@2` and `deploy-target@1` (the deploy
target realises the pool from `services.auth`). It contributes `services.auth` (`provider: cognito`,
`posture: private`, `sign_up: open`) and no Terraform. The module stays stdlib-only, because
`api/go.mod` is `api-go`'s.

## The environment

All of these come from the auth module's `environment` output; **if any is missing or malformed, every
`/api` route but `/api/health` answers 503** (fail closed; a partial set is never a fallback).

| Variable | Used for |
| --- | --- |
| `AUTH_ISSUER` | the `iss` every token must carry; its host is where `DeleteUser` is called |
| `AUTH_CLIENT_ID` | the `aud` of an ID token (`client_id` of an access token); the public client |
| `AUTH_JWKS_URL` | the pool's signing keys |
| `AUTH_POSTURE` | `private` or `public`; both are accepted and, in 0.1.0, both mean private: every route needs a person. Per-route public endpoints are a later feature. A deployed function never acts for the anonymous owner |
| `AUTH_LOGIN_URL` | the authorize request, less `state` and the PKCE parameters, which the API appends. Its `scope` and `redirect_uri` are used as written: the scope is the deploy's to grant. The token and revoke endpoints are the scheme and host of this URL plus `/oauth2/token` and `/oauth2/revoke` |
| `AUTH_LOGOUT_URL` | Cognito's sign-out page, which returns to the site |
| `SIGNUP_DOMAINS` | optional: email domains allowed to sign in (below). A malformed entry also closes the API |

A local run (`ALBITOR_LOCAL=1`) uses `api-go`'s local issuer instead, as `auth-cognito` does, and never
beside the variables above or on a deployed function. There the hosted routes say they are not available.

## The flow

1. `GET /api/auth/login?next=` makes a random state and a PKCE verifier, keeps the state, the verifier
   (with its issue time) and `next` in three short-lived `__Host-` cookies (HttpOnly, Secure, SameSite=Lax,
   Path=/, 10 minutes), and redirects to `AUTH_LOGIN_URL` plus `state`, `code_challenge` and
   `code_challenge_method=S256`. `next` is kept only if it is a path on this site: a scheme, host, `//`,
   a backslash, a control character, `/api/...` or one of the sign-in screens becomes `/`.
2. `GET /api/auth/callback` checks the state against the cookie in constant time, refuses a missing or
   over-old verifier, exchanges the code at `/oauth2/token` (form post: `client_id`, `code_verifier`,
   `redirect_uri`; no secret), verifies the ID token (signature, `iss`, `aud`, `token_use`, expiry),
   requires `email_verified`, applies the domain rule, then sets the session.
3. Any failure redirects to `/sign-in?error=<code>`; the sign-in page shows it and does not redirect
   again. A domain refusal redirects to `/not-allowed` (no address in the URL) and sets no session. A
   cookie value over 4000 bytes is refused (`toolarge`) rather than dropped by the browser, which would
   loop. As a last guard against a loop the cause of which is not known (a browser that blocks the
   cookies), the sign-in page hands the browser to Cognito at most once per 30 seconds per tab; a
   second arrival says "We could not keep you signed in".
4. An expired session is refreshed from the refresh cookie at the token endpoint (`refresh_token` grant).
   A refused refresh token ends the session; an unreachable endpoint is a 503 and keeps it.
5. `POST /api/auth/sign-out` revokes the refresh token (`/oauth2/revoke`, best effort), clears the cookies
   and answers `{"logout_url": ...}`. The sign-out page navigates there, so Cognito's own cookie is
   cleared too; without it the next "Sign in" would skip the password. Sign-out revokes the refresh
   token, but an ID token already issued stays valid until it expires (at most an hour).
6. `POST /api/auth/delete` is the person deleting their own account (below).

Both POSTs need `Content-Type: application/json` (415 otherwise), so another site's HTML form cannot
sign anyone out or delete anything. That is the same defence `auth-cognito` uses, with the Strict cookies.

### The cookies

Every cookie is a `__Host-` cookie: Secure, `Path=/`, no `Domain`. Every app is a subdomain of one
shared domain, so a sibling app could otherwise plant a cookie of the same name for this one; a browser
refuses to accept a `__Host-` cookie from anywhere but this host. The refresh cookie is therefore sent
with every request to the site, not only the API's. A refresh that returns a new refresh token
replaces the cookie.

### Why the callback does not redirect (the Strict cookie)

The session cookies are `SameSite=Strict`. The callback is reached by a cross-site top-level navigation
(from Cognito). A browser **stores** a Strict cookie set on that response, but does **not send** it on
any request whose navigation chain began on another site, and the 302 the callback would make to `next`
is such a request: the first page load would arrive with no session and the guard would send the person
back to sign in. Measured in Chromium: after a cross-site navigation, a 302 to `/me` arrives with no
cookie, and a 200 page with `<meta http-equiv="refresh" content="0;url=/me">` arrives with it.

So the callback answers a **200 HTML page** that navigates itself with a meta refresh (and a link, for
a browser that will not). A navigation started by a page of this site is same-site. The page needs no
script, so a strict Content-Security-Policy does not block it. The cookies stay Strict; nothing falls
back to Lax. (The transient login cookies are Lax on purpose: they must be sent on the return
navigation. They hold no session.)

### Who may hold a session (`email_verified` and `SIGNUP_DOMAINS`)

The client is public, so anyone can run the authorization code flow themselves and present the tokens
to the API. The rule is therefore not only at the callback: one function, `acceptIDToken`, is applied
to **every** ID token the API takes: at the callback, on every refresh, and on **every request**, whether
the token arrives as a bearer header or a session cookie. It must be an ID token (the gate accepts ID
tokens only, never an access token, which carries no email), the address must be present and verified,
and `SIGNUP_DOMAINS` must allow it. The address is trimmed and lower-cased once, there.

`SIGNUP_DOMAINS` is a comma- or space-separated list of `example.org` (exactly that domain) or
`*.example.org` (any subdomain, not the bare domain). Unset or empty means any verified address. A
malformed entry closes the API. Cognito's sign-up page cannot apply it, which is why it is here. A
person can change their own email to another verified address, so it is the token's address, every
time (a string comparison, no call). At the callback a refusal goes to `/not-allowed` and sets no
session; elsewhere it answers `401 email_not_allowed` and clears the session cookies, and the route
guard sends the person to `/not-allowed`. Cognito still holds the account of a refused person;
removing it is the operator's.

### Account deletion

Cognito's `DeleteUser` takes the person's **access** token and needs the `aws.cognito.signin.user.admin`
scope on it. The API holds no access token (the session is the ID token), so it first trades the refresh
cookie for a fresh one (which also means a stale session cannot delete an account), checks the scope,
and calls `DeleteUser`. A token without the scope is refused up front with `403 scope_missing` and a
clear message; Cognito's own refusal of the scope reads the same. **The deploy must grant the scope**
(it is the module's `AUTH_LOGIN_URL` scope; this plugin never adds one). On success the cookies are
cleared and the logout URL is returned.

To remove an app's own data for the person, set `shell.OnAccountDeleted` from the app's code (an `init`
in a file of its own, never an edit to the managed file). It runs once after the account is deleted with
the session's identity. The person cannot retry (the account is gone), so a failure is logged, not
returned: make it idempotent and keep an erasure job as the backstop.

## Requirements on the deploy target

- **Hosted login must be on** (the module's site is set), so `AUTH_LOGIN_URL` and `AUTH_LOGOUT_URL`
  exist, with the callback `<site>/api/auth/callback` and logout `<site>/` registered on the client.
- **The route `/api` must reach the API** (CloudFront does), so `/api/auth/callback` is the API's.
- **Account deletion needs the `aws.cognito.signin.user.admin` scope** on the client and in
  `AUTH_LOGIN_URL`.

## Known gaps

- Two tabs that start a sign-in at once share one set of transient cookies: the later tab's state
  replaces the earlier's, so the earlier tab's callback fails (`/sign-in?error=failed`) and the person
  continues from there. Nothing is signed in wrongly.

- No "You have signed out" or "Your account has been deleted" banner on the home page: Cognito returns to
  a fixed `<site>/`, and the home page is the app's. An app with every page private sends a signed-out
  visitor straight back to Cognito's sign-in; give the app a public landing page if that is not wanted.
- The refusal page's "contact" line is generic; reword it for the service.
- Logic ships as managed files, not a versioned module, so a fix reaches an app on a compose or upgrade.

## Verification

`scripts/verify-auth.sh auth-hosted` (run by CI) lays this plugin over `api-go`'s tree, renders the router
with this plugin's slots, and runs `gofmt`, `go build`, `go vet` and `go test`: the flow against a fake
Cognito (JWKS, token, revoke and `DeleteUser`) and the local issuer. `tests/compose` runs the real
manifest parser, the `auth@1` conformance suite and the composer over this plugin in the default
profile, on every `component-kit@2` kit. `scripts/compose-check.sh --build` builds the composed web tree.
The journey needs a hosted pool, so it runs in a deployed run, where `test-playwright`'s `auth.setup.ts`
signs in on Cognito's page once.
