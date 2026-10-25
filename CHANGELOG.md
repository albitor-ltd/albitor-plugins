# Changelog

All notable changes to the Albitor plugins catalogue. Plugin code changes are recorded in each plugin's own repository; this log tracks catalogue membership and the pinned versions.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Fixed

- `gds` `0.1.8` (albitor-ltd/albitor-plugins#68): GOV.UK branding (crown, GOV.UK logotype, OGL,
  Crown copyright, GDS Transport) now keys on the build's explicit "on GOV.UK" statement, never on
  the domain or audience. Not stated or "not on GOV.UK" gives the crownless pattern (organisation
  name header, system font stack). Skill, auditor agent and references updated consistently.

### Added

- `auth-hosted` `0.1.0` (new, untagged): an `auth@1` provider for the hosted lane. The hosted-app auth module's
  Cognito client is public and allows only the authorization code flow, so `auth-cognito` (password sign-in, needs
  `AUTH_USER_POOL_ID`) fails closed there. `auth-hosted` reads the module's `AUTH_*` environment, redirects to the
  hosted sign-in with PKCE, finishes the callback (state, code exchange, ID token and `email_verified` checks), and
  keeps the session in HttpOnly SameSite=Strict cookies; the callback answers a 200 page that navigates, not a 302,
  because a browser does not send a Strict cookie on a redirect that follows a cross-site navigation. Also sign-out
  through Cognito's logout, self-service account deletion (needs the `aws.cognito.signin.user.admin` scope), and an
  optional `SIGNUP_DOMAINS` rule, which with `email_verified` is applied to every ID token the API takes (callback, refresh and each request, bearer or cookie), because the client is public. Cookies are `__Host-` cookies. `auth-cognito` is unchanged. The catalogue is not regenerated:
  it gains `auth-hosted` when the release tag exists.
- `test-playwright` `0.3.0`: the managed `auth.setup.ts` also signs in through a hosted sign-in page. After
  `/sign-in` loads, a landing on a host other than the app's with the Cognito classic hosted UI's username field
  visible (managed login v1) is signed in there: it fills the visible `input[name=username]` and
  `input[name=password]`, clicks the visible `input[name=signInSubmitButton]`, and waits for a page on the app's
  host that is neither `/api/auth/*` nor the sign-in route, failing fast if Cognito shows its error message. It
  then requires a cookie for the app's host that was not there before the submit (the login route's transient state cookies and cookies set on Cognito's host do not count; on the hosted path it must also look like a session, by default a `__Host-` cookie that is not a transient one, overridable with `ALBITOR_SESSION_COOKIE_PATTERN`). An
  in-app email field is always taken as the in-app form, whatever the host. The in-app path is unchanged. The
  silent-pass hole is closed: a demo password in hand now makes a missing sign-in screen fail the setup, and a
  deployed run (`ALBITOR_JOURNEY_BASE_URL` or `PLAYWRIGHT_API_ORIGIN`) with no password fails when the app
  declares `services.auth` in `albitor.yaml`, sets `ALBITOR_SIGN_IN_SCREEN=1` or renders a `SignInPage`. Only an
  app with none of those (auth-none), or a local run with no password, still runs signed out. New
  `helpers/hosted-signin.ts` with unit tests, and end-to-end tests of the hosted path against fake hosts. Not yet pinned in the catalogue (needs the `test-playwright/v0.3.0` tag).
- `test-playwright` `0.2.0` (albitor-ltd/albitor-plugins#63): the managed `auth.setup.ts` signs in by role and
  label, not test ids, because the standard sign-in screen is the component-kit@2 `SignInPage`, which carries
  none: a field labelled `Email...`, a field labelled `Password`, a button named `Sign in`. The "a password
  field but no email field" defect check is kept in that form. The source scan that makes the setup strict now
  looks for `SignInPage` (outside `web/src/lib/kit`) or the old `sign-in-email`. Migration: an app whose sign-in
  page still carries only the old test ids and unlabelled fields must label them, or stay on 0.1.2. Companion
  albitor change: the test-id contract in `docs/reference/web-app-architecture.md` and runner testdata.
  Signed in now also means the page has left the sign-in route: after `goto`, the setup records the sign-in
  URL (so a redirect to `/login` is the route) and waits, with `waitUntil: "commit"`, for the pathname (the
  pathname plus hash under hash routing) to change. A same-route challenge page, such as a new-password
  step, fails the setup instead of saving a session that is not one.

- `auth-cognito` `0.3.0`, `kit-shadcn` `0.5.0`, `kit-carbon` `0.5.0`, `kit-gds` `0.5.0`
  (albitor-ltd/albitor-plugins#63; needs albitor core with component-kit@2, albitor-ltd/albitor#3746): the
  three kits declare `provides: ["component-kit@1", "component-kit@2"]` (a `requires` resolves against an
  exact major, and web-svelte still requires `@1`). kit-shadcn and kit-carbon move `SignInPage.svelte` from
  `web/src/lib/kit/pages/` to `web/src/lib/kit/` (certification of a kit listing `@2` requires the file there,
  as kit-gds already had it); `pages/index.ts` still re-exports it. `auth-cognito` now
  `requires` `component-kit@2` and its `(auth)` screens no longer carry `when component-kit: kit-shadcn`, so
  they are seeded on every `@2` kit (kit-carbon and kit-gds included, proved in `tests/compose/auth_test.go`).
  Its sign-in screen renders the kit's `SignInPage`; the new-password step and the sign-up, confirm and
  reset screens use the kit's `Field` and `Button`.
  Migration (documented breakage): an app needs a kit at `0.5.0` before `auth-cognito` `0.3.0` will compose.
  The screens and `sign-in.spec.ts` are app-owned examples, so an existing app should keep its old sign-in page
  (with its `sign-in-*` test ids) and the old spec, though what compose does with example files on an upgrade
  is under review in albitor-ltd/albitor#3722, so check them after upgrading; to adopt the kit's `SignInPage`, take the new
  `(auth)/sign-in/+page.svelte` and `tests/journeys/sign-in.spec.ts` together (the spec now finds the form
  by role and label, as `SignInPage` takes no test ids). `SignInPage` gains an optional `email` prop (the
  email field's initial value) in all three kits, which auth-cognito fills from `?email=`. On kit-shadcn and kit-carbon, an app that already has
  `pages/SignInPage.svelte` should move it to `lib/kit/SignInPage.svelte`, then point both `index.ts` and
  `pages/index.ts` (and `pages/PagesShowcase.svelte`, which now imports from `../index`) at it; whether
  compose seeds the new file over an old app's `index.ts` is the question in albitor-ltd/albitor#3722, so check
  `web/src/lib/kit/` after upgrading.
- `kit-gds` `0.4.0`, `gds` `0.1.9`, `kit-carbon` `0.4.0` (albitor-ltd/albitor#3710): the
  preview-tokens contract gains an optional `roles` block (`header` and `header_withdrawn` as
  `background`/`foreground`, `link`, `focus`), so the structure preview draws the header band, link
  and focus colour. `kit-gds` and `gds` ship a neutral `header` (used off GOV.UK, where the buyer's
  brand colour paints the band instead; never the GOV.UK blue) and the GOV.UK blue as
  `header_withdrawn`, used only where the brand colour is withdrawn, and a `focus_ink` (#0b0c0c) so
  the focus indicator is the #ffdd00 fill with black text. `kit-carbon` ships its own
  roles as the non-GOV.UK proof. `validate-preview-tokens.sh` and its tests know the block.

- `kit-shadcn` `0.4.0`, `kit-carbon` `0.3.0`, `kit-gds` `0.3.0` (albitor-ltd/albitor-plugins#62): ship
  the `component-kit@2` standard shell contract (ADR-0095 section 6) as exports; they
  still declare `provides: ["component-kit@1"]` because the platform registry rejects major 2. Each kit exports `AppShell`,
  `Footer` and `SignInPage` from `web/src/lib/kit/index.ts` with the same props: `AppShell` gains an
  optional `footer` snippet rendered after `<main>`, `Footer` is new (`links`, `children`; one
  `contentinfo` landmark), and `SignInPage` is exported from the kit root (new in kit-gds; re-exported
  from `pages/` in the other two). New managed `contract.test.ts`, `ContractProbe.svelte`,
  `signin.test.ts` and `SignInHarness.svelte`. `SignInPage` behaves the same in all three: the error
  stays until the next attempt's result and takes focus on every failed attempt, a repeated sentence
  included (kit-gds no longer hides it while `busy`); `Footer` keys its links by position, so two
  links to one address render. The contract is in the README.
  Migration: this is not a drop-in upgrade. The managed tests import `index.ts`, which is the app's
  after the first compose, so an existing app must add the exports to its `index.ts` (`Footer`,
  `FooterLink`, `SignInPage`) and the `footer` snippet to its `AppShell.svelte` (rendered after
  `<main>`), or `contract.test.ts` fails. Copy the kit's new `Footer.svelte`, and its changed
  `SignInPage.svelte` (kit-shadcn and kit-carbon: `pages/SignInPage.svelte`, whose error-focus
  effect changed, so `signin.test.ts` fails against the old one; kit-gds: new), from the kit's
  `templates/web/src/lib/kit/`. They are `example` files; what compose does with them on an upgrade
  is under review in albitor-ltd/albitor#3722, so check your `web/src/lib/kit/` files after
  upgrading rather than relying on either behaviour.

- `kit-gds` `0.2.0` (albitor-ltd/albitor-plugins#65, albitor-ltd/albitor#3707): fills `web.root-classes`
  with `govuk-template` (html) and `govuk-template__body` (body). `AppShell` adds `js-enabled` and
  `govuk-frontend-supported` to the body while it initialises, before children mount (instead of Frontend's inline script, so no CSP
  exception) and removes only those on destroy; new managed `rootclasses.test.ts` and fixture. Needs `web-svelte`
  0.4.0; merge after it. Still no Crown or GOV.UK logotype.

- `web-svelte` `0.4.0` (albitor-ltd/albitor-plugins#65, albitor-ltd/albitor#3707): hosts the `web.root-classes`
  slot. `web/src/app.html` is now a managed `html` slot template with
  `{{albitor.slot.web.root-classes.html}}` on `<html>` and `{{albitor.slot.web.root-classes.body}}`
  on `<body>` (existing attributes kept), so a kit or style can set root classes. No entries renders
  `class=""`. `app.html` moves from app-owned to managed. Merge before any kit fills the slot.
  Migration: an app that edited `app.html` (favicon, meta tags, attributes) loses those edits on
  upgrade. Move head content to `<svelte:head>` in a layout; root classes go through `web.root-classes`.

- `kit-gds` `0.1.0` (albitor-ltd/albitor#3563): a `component-kit@1` plugin for Svelte that absorbs
  the `gds` pack. `AppShell` (text-only header, service navigation), `Button`, `Dialog` (a native
  `<dialog>`), `Field`, `ErrorSummary` and `Table` in GOV.UK Frontend 6.4.0 markup and classes,
  compiled with `sass` from `govuk-frontend@6.4.0`, with a `:where()` base layer so bare elements
  read as GOV.UK and one brand-colour token for apps off GOV.UK; no Crown, no GOV.UK logotype, no GDS Transport font. `themeable:
  false`. Managed axe, dialog and contrast tests (colours read from the installed stylesheet), the
  `govuk-design-system` and `govuk-accessibility` skills (each `metadata.capability:
  design-system`, with the audit and statement procedures as references) and a
  `preview-tokens.json`. `plugins/gds` is unchanged; its retirement is separate. New and untagged,
  so not in the catalogue until released.
- CI compose matrix: every component-kit plugin is composed with every visual-language plugin on
  each certified profile (`tests/compose/kit_look_test.go`, albitor#3568). A themeable kit must
  compose with the look; a kit that is not (`kit-carbon`) must be refused as `kit_not_themeable`,
  the answer the platform gives before it places an app's look on the socket. No plugin changed.

### Changed

- Go 1.26 becomes 1.27 (toolchain go1.27.3), `api-go` `0.1.2`: the template `api/go.mod` is `go 1.27`, so
  every generated app builds on Go 1.27. This repository's CI `setup-go` jobs, `tests/compose/go.mod` and the
  `compose-check.sh` workspace move with it. The other plugins' workflows read `api/go.mod` and follow. No
  golangci-lint is pinned here. Needs the `api-go/v0.1.2` tag before the catalogue can pin it.
- Dependency refresh (albitor-ltd/albitor#3823): `web-svelte` `0.4.1`, `kit-carbon` `0.5.1`, `kit-gds`
  `0.5.1`, `kit-shadcn` `0.5.1`, `test-playwright` `0.3.1`, `deploy-byoa-aws` `0.3.4`, `deploy-hosted`
  `0.1.2`. The `deploy-byoa-aws` root's module `ref`s follow to `deploy-byoa-aws/v0.3.4`. No Go line
  changes (api-go `go 1.26` stays).
  - Node 24.21.0 becomes 24.22.1 in every template and in this repository's CI; the `test-playwright`
    journeys workflow moves from Node 22 to the same pin. `web-svelte` `@types/node` 22.20.4 to 24.19.2.
  - Terraform CLI 1.16.4 to 1.17.0 (CI and the `deploy-byoa-aws` ci, deploy and invite workflows); the
    AWS provider 6.66.0 to 6.70.0 in the five modules, the root and its lock (regenerated for linux and
    darwin, amd64 and arm64).
  - Actions, still full-SHA pinned: checkout v7.0.1, setup-go v7.0.0, setup-node v7.1.0,
    upload-artifact v7.0.2 and download-artifact v8.0.2 (a matched pair), setup-terraform v4.0.1 and
    configure-aws-credentials v6.3.0 (already current). No `with:` block needed a change: every
    checkout keeps `persist-credentials: false`, no step uses `pull_request_target`, no `setup-node`
    caching is configured (v5 and later cache npm automatically only when `package.json` carries a
    `packageManager` field, which no template does), the artifacts are directories of many files so
    the v7 `archive` default is unchanged, and the artifact names are fetched by name.
  - npm pins stay under the 14-day rule (newest release at least 14 days old on 2026-10-25), so
    some requested latest versions are held back: `@sveltejs/kit` stays 3.0.1 (3.1.0 is from 2026-10-14),
    `@playwright/test`/`playwright-core` 1.64.0 (1.64.1 is from 2026-10-14; the unit test that guards
    the rule now allows 1.64.0), `@axe-core/playwright` stays 4.13.0, `govuk-frontend` 6.5.1 (6.6.0 is
    from 2026-10-15; the kit markup is unchanged and 6.5.x only adds trial components), `bits-ui`
    stays 2.19.3, `sass` stays 1.105.2. Moved: svelte 5.57.2, vite 8.3.4, vitest 5.0.3,
    `@tanstack/svelte-query` 6.3.2, `axe-core` 4.14.0, jsdom 30.1.2, `carbon-components-svelte`
    0.113.1, `carbon-icons-svelte` 13.16.0, `@ibm/telemetry-js` 1.12.0. `carbon-components-svelte`
    0.113 breaks the `Disclosure` `toggle` detail (now `{ open: boolean }`) and drops unused IBM Plex
    weights (100, 200, 450, 500, 700); the kit uses neither (weights 400 and 600 only). Two other 0.113
    changes needed kit-carbon fixes: its SideNav calls `scrollIntoView`, which jsdom lacks, so the kit
    now ships a managed `web/src/lib/kit/vitest.setup.ts` (stub) that its `vitest.config.ts` loads, or any jsdom test
    that renders the shell fails with an unhandled rejection (a new managed file: an existing app gets
    it and the changed `vitest.config.ts` on the next compose); and `all.css` now writes the dark
    themes as `:root[theme=g100]{...}` and omits tokens identical to `:root`, which `contrast.test.ts`
    now reads.
    TypeScript stays 6.x. The two template lockfiles are regenerated.
  - Release tags needed after merge: `web-svelte/v0.4.1`, `kit-carbon/v0.5.1`, `kit-gds/v0.5.1`,
    `kit-shadcn/v0.5.1`, `test-playwright/v0.3.1`, `deploy-byoa-aws/v0.3.4`, `deploy-hosted/v0.1.2`; then
    regenerate the catalogue pins.

- `kit-shadcn` `0.3.0` (albitor-ltd/albitor#3563): the kit's skill takes over the `shadcn`
  pack's design-system role. `SKILL.md` declares `metadata.capability: design-system`, the plugin
  gains the `design-system` keyword (so `DetectKind` lists it as a design system), and a new
  `references/design-system.md` carries what still applies to a Svelte-first kit: the Bits UI
  primitive for each shadcn component, variants, focus, contrast traps and icons. The accessibility
  statement and VPAT/ACR guidance from `shadcn-accessibility` moves into
  `references/accessibility-statement.md`, and the framework-neutral patterns (human labels,
  document title, async status, input types, command palette) into `references/patterns.md`. The
  React and Tailwind content (CLI, `cn()`, `cva`, Radix, `components.json`) and the rest of
  `shadcn-accessibility` (WCAG 2.2 and testing, which `test-playwright` covers) are not carried. `shadcn` is untouched until
  it is retired. Untagged, so the catalogue still pins the previous version until released.
- `test-playwright` `0.1.2` (albitor-ltd/albitor#3588): `journeyStep` photographs the whole page
  rather than the viewport, including `h-screen` shells that scroll an inner container (the
  viewport is temporarily grown to the needed height, then restored). The image is capped at 6000
  CSS px tall (`scale: "css"`, so device scale factor cannot inflate it past the runner's 8 MB
  per-file cap); a taller page is clipped and its ledger line gains `clipped: true`. Ledger
  `width`/`height` are read from the PNG itself. Without a viewport or measurement it falls back
  to the plain viewport shot, never an uncapped full page. Untagged, so the catalogue still pins
  the previous version until released.
- `shadcn` `0.1.6`: secondary text passes AA on the muted surface. `text_secondary` in
  `preview-tokens.json` and the preview page's light `--muted-foreground` move from zinc-500
  (`#71717a`, 4.40:1 on `#f4f4f5`) to zinc-600 (`#52525b`, 7.03:1). Dark theme unchanged (5.81:1).
  Untagged, so the catalogue still pins the previous version until released.
- `kit-shadcn` `0.2.1`, `look-business`, `look-startup` and `look-consumer` `0.1.1`: mobile
  navigation guidance. The kit describes the bottom tab bar and the menu-button drawer (when to use
  which, a "More" overflow, toggle accessibility, 44px targets, safe-area inset); each look states
  its phone navigation preference. Guidance only; no tokens or components changed. Untagged, so the
  catalogue still pins the previous versions until released.

### Added

- `look-business`, `look-startup` and `look-consumer` `0.1.0` (albitor-ltd/albitor#3539): the three
  archetype looks, each a `visual-language@1` plugin with a `preview-tokens.json` (tokens from the
  approved looks mockup v4), a generic skill on applying the look (type, density, shape, dark mode,
  the archetype it suits) and a seeded `web/src/lib/look/look.json`. Baseline is the existing
  `shadcn` default; GOV.UK is not a look. Startup's `layered_shadow` is `elevation: "layered"` and
  `-apple-system` is dropped from the font stacks, as the platform contract has neither. New and
  untagged, so not in the catalogue until released.
- CI validation of every pack's `preview-tokens.json` (`scripts/validate-preview-tokens.sh`, with
  refusal tests in `scripts/test-preview-tokens.sh`), mirroring the platform's strict
  `governance.PreviewTokens` contract: unknown keys, bad lengths, colours and enums are refused
  (albitor-ltd/albitor#3539). No plugin changed.
- `look-consumer`: `dark_target: 5.0` added to `preview-tokens.json` and the seeded `look.json` (kept
  identical), so its dark mode meets the platform's contrast target. CI now also checks each look
  pack's seeded `look.json` equals its `preview-tokens.json`.

### Removed

- `catalogue/retired-entries.json` (albitor-ltd/albitor#3176, #3530): the five retired `visual-*`
  entries are no longer carried through the legacy catalogue mode, which is now generated from
  `catalogue/legacy-pins.json` alone. The monorepo catalogue is unchanged.

### Fixed

- `deploy-byoa-aws` `0.3.2` (albitor-ltd/albitor#3426): the site edge passes the preview floor.
  Its own response-headers policy (HSTS, CSP, frame DENY, nosniff, strict referrer) sits on the
  site and `/api/*`; HSTS `includeSubDomains` and `preload` stay off unless set, as in 0.3.1. The
  SPA fallback is a viewer-request CloudFront Function on the site behaviour in place of the
  distribution-wide `custom_error_response`, so `/api/<unknown>` and a missing asset no longer
  return the shell (a route whose last segment has a dot is now treated as a file). The root stack
  hashes the built shell's inline scripts into `script-src` (never `'unsafe-inline'`; a site with
  no build is refused); the deploy workflow copies the HTML the bucket serves into the apply, which
  admits that live shell too while the new bundle ships, now before the seed. The example
  `terraform.tfvars` no longer cites an ADR.
- `deploy-byoa-aws` `0.3.3` and `data-dynamodb` `0.1.1` (albitor-ltd/albitor#3437): the rendered
  templates carry no internal references. `deploy.yml`, the root `variables.tf`,
  `seed-platform-accounts.sh` and the store's `sigv4.go` drop the issue numbers, ADR ids and core
  repository slug their comments cited, keeping the reasons. `ALBITOR_APP_SECRETS_SEGMENT` keeps its
  name: the platform writes it into the deploy environment file, and the core gate accepts it as an
  identifier the pipeline resolves. `scripts/validate-plugins.sh` now refuses a template that cites
  an issue, an ADR or the core repository.

### Changed

- `web-svelte` `v0.3.0`: the managed root layout passes the platform's `app_name` token to the
  kit's `AppShell` as `title`, so the header names the app instead of "App"
  (albitor-ltd/albitor#3426). Needs an albitor composer that owns `app_name`
  (albitor-ltd/albitor#3428); the `component-kit@1` contract now names `title: string`, which
  `kit-shadcn` and `kit-carbon` already take.

- CI certifies every plugin into every certified profile with the platform's harness, `byoa-aws`
  included: composed `albitor.yaml` against the deploy target's `renders`, and `terraform validate`
  over each composed root (albitor-ltd/albitor#3333). `deploy-byoa-aws` and `deploy-hosted` now
  declare `renders` in `plugin.json`; released as `deploy-byoa-aws` `v0.3.1` and `deploy-hosted` `v0.1.1`.
- Repository becomes the first-party plugin monorepo (albitor-ltd/albitor#3158): `plugins/<name>/`
  for auth, carbon, shadcn, mui, gds and docs, imported from each repo's `main`. The committed
  catalogue still points at the old repositories and tags until the switch-over (see README).
- Catalogue is generated by `scripts/build-catalogue.sh`; CI added.
- Switch-over (albitor-ltd/albitor#3190): the committed catalogue is now `--mode monorepo`, one
  `git-subdir` entry per plugin pinned to its `<name>/vX.Y.Z` tag and that tag's commit (`sha`).
  Pins at the switch, all tagged from one commit: `auth` `v0.7.0`, `carbon` `v0.1.5`, `shadcn`
  `v0.1.5`, `mui` `v0.2.2`, `gds` `v0.1.7`, `docs` `v0.2.0`, `api-go` `v0.1.0`, `web-svelte`
  `v0.1.0`. Pin history since `auth` `v0.5.0`: `auth` `v0.5.0` to `v0.7.0`; `carbon` `v0.1.4` to `v0.1.5`; `shadcn` `v0.1.4` to `v0.1.5`; `mui` `v0.2.1` to
  `v0.2.2`; `gds` `v0.1.6` to `v0.1.7`. The retired `visual-*` entries leave the catalogue.
- CI: `cancel-in-progress` applies to pull requests only; `validate-plugins.sh` requires a legacy
  pin only with `--mode legacy`; `build-catalogue.sh` fails on a missing version or tag; the
  catalogue description must equal `plugin.json` (it had drifted for `auth`, `mui`, `docs`).
- `kit-shadcn` states `"themeable": true` and `kit-carbon` `"themeable": false`; the `albitor-kit`
  layer is documented as `kit-shadcn`'s theming contract (albitor-ltd/albitor#3236). `web-svelte`'s
  npm dependencies are exact pins, and its seed `web/package.json` and lockfile match them
  (albitor-ltd/albitor#3235). `@sveltejs/kit` 3.0.1 (2026-10-06) and `@sveltejs/adapter-static`
  4.0.0 (2026-10-01) are pinned under the 14-day rule because no older release of their major
  exists; transitives in the seeded lockfile are not subject to the rule (current practice).
  `compose-check.sh` runs `npm install` in the composed `web/`, as the lockfile is the app's, and
  CI pins Node 24.21.0 (npm 11), as npm 10.9 crashes on a lockfile-less install
  (albitor-ltd/albitor#3178).
- Release `kit-shadcn`, `kit-carbon` and `web-svelte` `v0.1.1`: the themeable markers, `web-svelte`'s
  exact npm pins and the `tokens.css` and README changes above (#20) were merged without a version
  bump, so they have no tag. The catalogue pins them once `<name>/v0.1.1` exists.
- Release `web-svelte` `v0.1.2`: the build command is `npm install`, as the lockfile is the app's
  (#23). The catalogue pins it once `web-svelte/v0.1.2` exists.
- Release `test-playwright` `v0.1.1`: the workflow installs the app's `web/` with `npm install` and the
  skill says the same (#23 changed the template without a version bump, so `v0.1.0` still has
  `npm ci` there and fails once the composer drops `web/package-lock.json`). `compose-check.sh` now
  lints every shipped workflow: an `npm ci` step needs a lockfile shipped in its working directory.
  The catalogue pins it once `test-playwright/v0.1.1` exists.
- Release `deploy-byoa-aws` `v0.1.1` (albitor-ltd/albitor#3178): `deploy.yml` declares the
  `environment` and `ref` dispatch inputs the platform sends for a promotion, runs in the named
  GitHub Environment, and gains the `ship SPA` job that stamps the `deploy-receipt` notice the
  platform reads. The catalogue pins it once `deploy-byoa-aws/v0.1.1` exists.
- Release `deploy-byoa-aws` `v0.2.0` (albitor-ltd/albitor#3174, #1161): `services.auth` is rendered
  instead of refused. A new `auth` module makes a Cognito user pool (email as the username,
  `invite_only` or `open` sign-up, deletion protection), a public app client (`USER_PASSWORD_AUTH`,
  `PreventUserExistenceErrors` on) and one group per `groups` entry; the function is handed
  `AUTH_USER_POOL_ID` and `AUTH_CLIENT_ID`, and the root outputs `auth_user_pool_id` and
  `auth_client_id`. Usernames are case-insensitive. Module and root golden tests cover it. (The
  pool's empty start, which v0.2.0 left to an operator, is closed in `v0.3.0` below.) The catalogue pins it once `deploy-byoa-aws/v0.2.0` exists.
- Release `deploy-byoa-aws` `v0.3.0` and `auth-cognito` `v0.2.0` (albitor-ltd/albitor#3354): nobody has to
  create a user by hand any more. Every `dev` deploy seeds the owner's demo account and Albitor's
  verifier account into the pool (`scripts/seed-platform-accounts.sh`, a step of `deploy.yml` between
  the apply and the ship, under the deploy role, passwords read from the app's secret store; a
  promotion seeds nothing), and a new `invite` workflow plus `scripts/invite-user.sh` lets the owner
  invite anyone by email. `auth-cognito` answers the `NEW_PASSWORD_REQUIRED` challenge an invited
  person meets at first sign-in: `409 new_password_required` carries the challenge session,
  `POST /api/auth/new-password` finishes it, and the sign-in screen asks for a password of their
  own. The receipt carries `auth_user_pool_id`, `auth_client_id` and `seed_status` (#3022); the seed
  reads the passwords under the path segment Albitor writes as `ALBITOR_APP_SECRETS_SEGMENT` in
  `env/<environment>.deploy.env` and fails loudly without it; the pool's invitation email names the
  app and the sign-in address; the sign-in screen moves focus to the new-password step and states the
  password rules; `docs/inviting-people.md` is delivered to the owner at handover. The README's "operator path" and its known-gap line are removed. The catalogue pins
  them once `deploy-byoa-aws/v0.3.0` and `auth-cognito/v0.2.0` exist. `deploy-hosted` needs no change: the
  hosted pipeline seeds both accounts on a preview deploy.
- Release `web-svelte` `v0.2.0` (albitor-ltd/albitor ADR-0090, amendment of 2026-10-15): the
  `client-data@1` socket is retired and both data layers move into `web-svelte`, the online layer
  under `src/lib/data/` and the offline-first `@albitor/sync` layer under `src/lib/sync/`, file for
  file, with their tests and skills (`data-online`, `data-offline`). The platform's `data_layer`
  token (`online` or `offline`, from the plan's `offline` need) selects the entry point
  (`src/lib/data-layer/index.ts`), which `web-svelte` names in its own `web.providers` fragment as
  `DataProvider`. `@tanstack/svelte-query` 6.3.0 and `fake-indexeddb` 6.2.5 join the seeded
  `web/package.json` and lockfile. Needs an albitor composer that owns `data_layer`; an older one
  refuses it (`unknown_token`). The catalogue pins it once `web-svelte/v0.2.0` exists.
- Release `api-go` `v0.1.1`, and in `web-svelte` `v0.2.0` too: slot templates are declared in
  `plugin.json` as `mode: "slot-template"` entries (ADR-0090 amendment of 2026-10-14, items 1 and
  8), with their bodies at the generated paths under `templates/`
  (`api/internal/shell/router.go`; `web/src/lib/shell/nav.generated.ts` and
  `providers.generated.ts`). The `slots/slots.json` and `slot-templates.json` sidecars are removed:
  the platform never read them, so dev build 63716a5c refused to compose with four
  `no_slot_template` problems (`web.providers`, `web.route-guard`, `api.middleware`, `api.routes`).
  `tests/compose` now refuses a sidecar and composes every certified profile
  (`certification.Profiles()`) from the real plugin trees, with both data layers and every
  alternative plugin here, so this class cannot pass CI again.

- `kit-shadcn` and `kit-carbon` `v0.2.0` (albitor-ltd/albitor#2809, ADR-0090): page templates.
  Example files `ListPage`, `DetailPage`, `FormPage`, `DashboardPage`, `SettingsPage`/`SettingsSection`
  and `SignInPage` over `PageHeader` and `StateNotice` in `web/src/lib/kit/pages/`, and managed
  `PagesShowcase.svelte` and `pages.test.ts` that run axe over every template in every state
  (loading, failed, empty, busy, with errors, saved). The form and settings error summary takes
  `{ field?, message }` entries and links each to its control (WCAG 2.4.3, 3.3.1); `Field` accepts
  an explicit `id` for that. A new file set with no change to existing files except `Field`'s
  optional `id`. The catalogue pins them once `kit-shadcn/v0.2.0` and `kit-carbon/v0.2.0` exist.

### Removed

- `client-online` and `client-offline`: folded into `web-svelte` `v0.2.0` with the `client-data@1`
  socket. Their `v0.1.0` tags stay; the catalogue drops them when it is next regenerated.

### Added

- `web-svelte` plugin (albitor-ltd/albitor#3165): the `web-framework@1` SvelteKit shell, the first
  schema-2 plugin. Monorepo-only: left out of the legacy catalogue until the switch-over.
- `test-playwright` plugin (albitor-ltd/albitor#3175): the `test-harness@1` Playwright journey
  harness (watched browser context, `journeyStep`, one sign-in per suite, fixtures, walk mailbox),
  an axe WCAG 2.2 A/AA audit, a read-only managed `journeys.yml` workflow that passes the
  `test-harness` workflow lint, and the `wcag-22-aa` skill absorbing the accessibility reference the
  four design-system plugins each duplicate (they are unchanged). `scripts/compose-check.sh --journeys` and the compose CI job run the
  journeys against the composed default shell.
- `api-go` plugin (albitor-ltd/albitor#3168): the first schema-2 plugin, filling `api-runtime@1`,
  split from the starter's `_api`. A schema-2 plugin has no legacy pin, so it appears in the
  catalogue only in monorepo mode.
- `data-dynamodb` plugin (albitor-ltd/albitor#3169): the `data-store@1` plugin. A `Store`
  interface for the Go API over one DynamoDB table, the in-memory fake, the conformance suite
  (run against DynamoDB Local in CI) and the `services.tables` fragment. Stdlib only (the adapter
  signs its own requests). Enters the catalogue once tagged `data-dynamodb/v0.1.0`.
- `client-online` plugin (albitor-ltd/albitor#3171): the `client-data@1` plugin for online-only
  apps, schema 2. TanStack Query for Svelte behind `web.providers`, a shared retry policy, API query
  helpers and loading, offline, error and empty states. Not in the catalogue until it is tagged
  `client-online/v0.1.0` and the catalogue is regenerated.
- `client-offline` plugin (albitor-ltd/albitor#3172): the `client-data@1` plugin for offline-first
  apps, schema 2. `@albitor/sync`: an IndexedDB store per signed-in person, an outbox with
  in-flight-safe amendment, tombstones for deletions, single-flight cross-tab-locked sync, sign-out
  that refuses to discard unsynced work, auto sync, a status line, and tests proving each defect
  plantrain's sync layer had is absent. Not in the catalogue until it is tagged
  `client-offline/v0.1.0` and the catalogue is regenerated.
- `kit-shadcn` plugin (albitor-ltd/albitor#3166): the `component-kit@1` plugin for Svelte, a rewrite
  of `shadcn`, schema 2. App shell, nav, forms, tables, dialogs (Bits UI), toasts and empty states
  as app-owned example files over one token set that passes WCAG 2.2 AA contrast in light and dark,
  with axe and contrast tests that ship with the app, preview tokens and a Svelte-first skill. Not
  in the catalogue until it is tagged `kit-shadcn/v0.1.0` and the catalogue is regenerated.
- `auth-cognito` and `auth-none` plugins (albitor-ltd/albitor#3170): the split of `auth`, both
  `auth@1`, schema 2. `auth-cognito`: the API's identity seam (token check, `GET /api/me`, the
  sign-in calls, HttpOnly session cookies; stdlib only), a provider-independent session module and
  route guard, sign-in, sign-up, confirm, reset and sign-out screens seeded per kit (`kit-shadcn`
  first), a seeded sign-in journey, and the `services.auth` fragment. No Terraform and no SSM
  parameter: the deploy target realises the pool from `services.auth`. `auth-none`: a stub identity
  seam with one fixed owner and no screens. `auth` itself is unchanged and stays at `auth/v0.7.0`.
  Neither is in the catalogue until it is tagged `<name>/v0.1.0` and the catalogue is regenerated.
- `scripts/verify-auth.sh` and its CI job build, vet and test each auth plugin's Go over `api-go`.
- `kit-carbon` plugin (albitor-ltd/albitor#3167): the `component-kit@1` plugin for Svelte in the IBM
  Carbon style, a rewrite of `carbon`, schema 2, with the same shell pieces and props as
  `kit-shadcn`. Carbon's UI shell, forms, tables, `Modal` dialog and toasts as app-owned example
  files over carbon-components-svelte and carbon-icons-svelte, in Carbon's white and Gray 100
  themes, with axe and contrast tests (read from the installed stylesheet) that ship with the app,
  preview tokens and a Svelte-first skill. Not themeable by a visual-language plugin; the kit says
  so in its README, skill and a test, since `component-kit@1` has no manifest marker yet
  (albitor-ltd/albitor#3236). Not in the catalogue until it is tagged `kit-carbon/v0.1.0` and the
  catalogue is regenerated. Note: `carbon-components-svelte` runs an IBM telemetry `postinstall`
  from `@ibm/telemetry-js`, a transitive dependency declared as a range (`^1.5.0`) that floats
  past this repository's 14-day rule. The manifest pins `@ibm/telemetry-js@1.11.0` as a direct
  dependency (effective once the composer writes dependencies, albitor-ltd/albitor#3235), and
  `scripts/compose-check.sh` sets `IBM_TELEMETRY_DISABLED=true` and checks the pin. Albitor's build
  and certification environments need to set it too.

- Initial catalogue.
- `gds` plugin pinned at `v0.1.6` — GOV.UK Design System + WCAG 2.2 AA accessibility.
- `shadcn` plugin pinned at `v0.1.4` — shadcn/ui, Radix UI and Tailwind CSS.
- `mui` plugin pinned at `v0.2.1` — Material Design across frameworks.
- `carbon` plugin pinned at `v0.1.4` — IBM Carbon Design System.
- `auth` plugin pinned at `v0.4.0` — Cognito-backed sign-up and login.

### Changed

- `auth` plugin re-pinned to `v0.5.0` (2026-09-10) — optional SES-backed email sending for the
  user pool (`cognito_email_from`, `cognito_email_ses_identity_arn`); the default Cognito sender
  is capped at 50 emails a day per account.
- `docs` plugin pinned at `v0.2.0` — Diátaxis, C4, and arc42 documentation.
- `visual-editorial`, `visual-glass`, `visual-minimal`, `visual-neubrutal`, and `visual-swiss`
  plugins each pinned at `v0.1.0` — the five visual-language design layers.
