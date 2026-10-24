# test-playwright

The `test-harness@1` plugin
([ADR-0090](https://github.com/albitor-ltd/albitor) sections 2 and 13, albitor-ltd/albitor#3175).
The Playwright journey harness, composed into a new app's `web/tests/` by the platform's composer.
It comes from the starter's `web/tests` harness and the harness fixes made while building plantrain.

## What it provides

- **A journey harness** (`helpers/journey.ts`): `test` and `expect` over a browser context whose
  every page is watched from before its first navigation, and `journeyStep`, which judges each
  screen (an enforced Content-Security-Policy refusal, an uncaught exception, a `<title>` that is
  not a short name), photographs it and appends a ledger line when an automated walk asks for it.
- **Sign-in per auth socket** (`auth.setup.ts`, `helpers/local-auth.ts`, `helpers/storage-state.ts`,
  `helpers/credentials.ts`): one sign-in for the whole suite, the local test identities (`alice`,
  `bob`, `admin`) the api-runtime plugin's local issuer mints, and the platform walk's credentials
  file.
- **The sign-up mailbox** (`helpers/mailbox.ts`): how the one sign-up spec reads its confirmation
  email.
- **Fixtures and seed helpers** (`helpers/seed.ts`, `helpers/fixture-scope.ts`): records created
  through the app's own API and deleted again, exactly those and no others.
- **An axe audit** (`helpers/a11y.ts`, `a11y.spec.ts`): WCAG 2.2 A and AA, with reasoned waivers.
- **A smoke test** (`smoke.spec.ts`): the shell loads and the API keeps the edge error contract.
- **A read-only CI workflow** (`.github/workflows/journeys.yml`): `contents: read`, every action
  pinned to a commit, no secret, runs the unit tests and the journeys.
- Skills `test-playwright` (writing journeys) and `wcag-22-aa` (the WCAG 2.2 A and AA reference,
  absorbed from the four design-system plugins, which each carried a copy).

## Layout

```text
.claude-plugin/plugin.json   schema-2 manifest
templates/web/tests/**       composed into the app at the same paths
templates/.github/workflows/ the managed CI workflow
skills/                      test-playwright, wcag-22-aa
```

The harness is its own npm package (`web/tests/package.json` with a lockfile). The composer does not
merge dependencies into `web/package.json` yet (ADR-0090 amendment of 2026-10-14, item 7), and the
web-framework plugin owns that file, so the harness resolves `@playwright/test` and
`@axe-core/playwright` from `web/tests/node_modules`. The Playwright config lives in
`web/tests/playwright.config.ts`, not `web/`.

## Who may change what

Everything the plugin ships is `managed` (pinned by path and hash; builders cannot edit it) except
`web/tests/journeys/README.md`. Journey specs (`web/tests/journeys/`) and fixtures
(`web/tests/fixtures/`) belong to the spec author; the plugin writes nothing there.

## The workflow is the one sanctioned `.github/` file

A plugin that provides `test-harness` (or `deploy-target`) may write `.github/workflows/*.yml` in
`managed` mode. The conformance suite lints it at ingest under `DefaultWorkflowPolicy`: a top-level
`permissions` block with no write, no privileged trigger, every `uses:` pinned to a full SHA, only
`secrets.GITHUB_TOKEN`, no step that prints or uploads the environment. `tests/compose` runs that
lint over `journeys.yml` and proves each unsafe edit of it fails.

## Contracts it leaves to other plugins

- The auth plugin's sign-in screen is the component-kit@2 `SignInPage`, found by role and label, not
  test ids: a field labelled "Email...", a field labelled "Password" and a button named "Sign in". It
  may set `ALBITOR_SIGN_IN_SCREEN=1` for the run so the setup is strict about it (a front end that
  renders `SignInPage` outside the kit's own files, or names `sign-in-email`, makes it strict already).
  Signed in also means the page must leave the sign-in route: the setup records the URL the sign-in
  page loaded at and waits for its pathname (pathname plus hash under hash routing) to change, so a
  same-route challenge page, such as a new-password step, fails the setup rather than passing as a
  session. Redirect somewhere else on success (the app's landing page, `/`), even if it is a
  separate full-page load.
- The api-runtime plugin's local issuer lives at `/api/_local/auth` (`api-go` provides it).
- The web-framework plugin has an `npm run local` that serves the build on `PORT` with `/api`
  forwarded to `ALBITOR_API_ORIGIN` and `/healthz` answering (`web-svelte` does).

## Run it

From a composed tree, see the `test-playwright` skill. To check this plugin alone,
`scripts/compose-check.sh --build --journeys <albitor checkout>` at the repository root composes the
default shell, runs the harness's unit tests and runs the journeys in Chromium.
