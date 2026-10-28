---
name: test-playwright
description: How to write and run journey specs on the test-playwright harness - where specs go, the helpers to import, identity, fixtures, the accessibility audit and the local run. Use when a build writes or changes anything under web/tests, authors a journey spec, or runs the journeys.
---

# The test-playwright harness

Journeys are Playwright specs under `web/tests`. The harness is its own npm package
(`web/tests/package.json`, with its own lockfile), so it never touches the app's `web/package.json`.

## Who may change what

| Path | Whose | |
| --- | --- | --- |
| `web/tests/journeys/<journey>.spec.ts` | Spec author | One spec per journey; first line `// @journey <id>` |
| `web/tests/fixtures/**` | Spec author | Records the specs need, created through the app's own API |
| `web/tests/helpers/**`, `auth.setup.ts`, `playwright.config.ts`, `package*.json`, `smoke.spec.ts`, `a11y.spec.ts`, `roles.spec.ts`, `unit/**`, `.github/workflows/journeys.yml` | The plugin (managed) | Builders cannot edit them; report a needed change |

## Writing a journey

```ts
// @journey browse-claims
import { journeyStep } from "../helpers/journey";
import { expect, test } from "../helpers/seed";

test("browse the claims", async ({ page, fixtureScope }) => {
  const claim = await fixtureScope.create<{ id: string }>("/api/claims", {
    title: fixtureScope.uniqueName("claim"),
  });
  await journeyStep(page, "open the claims queue", async () => {
    await page.goto("/claims");
    await expect(page.getByTestId("claims-list")).toContainText(claim.id);
  }, { a11y: true });
});
```

- **Import `test` and `expect` from `helpers/journey` or `helpers/seed`**, never from
  `@playwright/test`. Every page in that context is watched from before its first navigation: a
  script the Content-Security-Policy blocks, an uncaught exception or a `<title>` that is not a short
  name fails the step. Only a sign-in or signed-out spec imports `@playwright/test`, to control its own
  identity.
- **One `journeyStep` per screen**, in the order a person reaches them, named for what the person is
  doing. Assert what the person must see. Pass `{ a11y: true }` to audit the screen for WCAG 2.2 A
  and AA (see the `wcag-22-aa` skill).
- **Find elements by stable test ids** (`getByTestId`), role or label. Never by CSS class, layout or
  position.
- **Seed through the app's API with `fixtureScope`**, which deletes exactly what the test created
  and fails the test if something is left behind. Never write to a database, and never put records in
  the app's code.

## Identity

- One sign-in for the whole suite: `auth.setup.ts` signs in once as the demo account and saves the
  session every spec inherits. Do not sign in again, and do not sign up.
- To be a role (a group of `services.auth.groups`), use the auth plugin's `auth@2` provider, never a
  hand-made user: `import { auth } from "./auth/provider"`, then `auth.roles()`,
  `await auth.seed(role, { request })` and `await auth.signInAs(role, { context, request, baseURL })`.
  Each role's account holds that group and no other. Locally it is an `api-go` local-issuer identity
  (`role-<group>`); deployed it is the deploy's seeded `<group>@albitor-role.invalid` signed in through
  the plugin's own sign-in. Use a signed-out context for it (`test.use({ storageState:
  EMPTY_STORAGE_STATE })`), not the journey fixture's `alice`. `roles.spec.ts` already proves every
  declared role signs in and holds only its own group.
- In a local run the journey fixture signs the browser in as the fixed `alice` identity from the local
  issuer, and `appAuthHeaders` carries the same bearer on direct `request` calls.
  The bearer goes only on the app's own fetch and XHR calls to its own origin, never to another host;
  The browser follows a redirect itself and that hop carries no bearer, so an app call that redirects
  (to another host or to another API path) arrives anonymous and fails closed. A streaming call
  (`Accept: text/event-stream`) is not intercepted, because intercepting buffers it, and so goes
  without the bearer: give it its own `Authorization` header (`appAuthHeaders`), which is always left alone.
- The demo password is typed through the page, not Playwright's `fill()`, and the field is emptied if
  the sign-in fails or times out (each step has its own short timeout), so no failure snapshot, report or trace holds it. Do not log it or fill it from a spec.
- Only the one sign-up spec signs up, with the platform's mailbox helper (`helpers/mailbox.ts`:
  `walkMailboxAvailable`, `walkAccount`, `awaitVerification`). It starts signed out
  (`test.use({ storageState: EMPTY_STORAGE_STATE })`), signs up with `walkAccount()`'s address and
  password exactly, and never deletes the account.
- `auth.setup.ts` signs in by role and label (the kit's `SignInPage`): a field labelled "Email...", a field
  labelled "Password" and a button named "Sign in". A hosted sign-in (redirect to another host, the Cognito classic hosted UI) is signed in there instead. An app
  with no sign-in (no password, no `services.auth`) runs signed out; one with a way in but no recognisable
  screen fails the setup. Specs that need a
  session skip themselves with `test.skip(!hasDemoSession(), ...)` in a `beforeEach`.

## Running

```sh
cd web && npm install && cd tests && npm ci && npx playwright install chromium
cd web/tests && npm test                 # boots the API and the web build, runs the journeys
cd web/tests && npm run unit             # the helpers' own unit tests
npx playwright test --list -c web/tests/playwright.config.ts web/tests/journeys/x.spec.ts   # from the repository root
```

`npm test` boots the stack under the local run contract (`api/`: `ALBITOR_LOCAL=1
ALBITOR_LOCAL_AUTH=1 go run .`; `web/`: `npm run local`) unless it is pointed elsewhere:

| Variable | Effect |
| --- | --- |
| `ALBITOR_LOCAL_BASE_URL` | The app is already booted here; boot nothing |
| `ALBITOR_JOURNEY_BASE_URL` | Browse this deployed app; boot nothing |
| `PLAYWRIGHT_API_ORIGIN` | Serve this web build against that deployed API |
| `LOCAL_WEB_PORT`, `LOCAL_API_PORT` | Ports (default 3000 and 8080) |

A spec that fails because the screen does not exist yet fails for the right reason: a missing route
or element, never a broken spec.
