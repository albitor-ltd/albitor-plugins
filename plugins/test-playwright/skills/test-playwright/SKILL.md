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
| `web/tests/fixtures/**` | Spec author | Records the specs need, created through the app's own API, and short real-speech media files |
| `web/tests/helpers/**`, `auth.setup.ts`, `playwright.config.ts`, `package*.json`, `smoke.spec.ts`, `a11y.spec.ts`, `roles.spec.ts`, `unit/**`, `.github/workflows/journeys.yml` | The plugin (seeded once) | Yours to change when the harness needs it; keep the audits and the sign-in checks |

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
  script the Content-Security-Policy blocks, an uncaught exception, a `<title>` that is not a short
  name, or developer source text on the screen (a visible line starting `//` or `/*`, or containing
  `<!--`, outside `<pre>`, `<code>`, a textarea, an input or an editable region) fails the step; user content that legitimately shows code in a plain
  element takes `data-allow-source-text` on its container to be skipped. Only a sign-in or signed-out spec imports `@playwright/test`, to control its own
  identity.
- **One `journeyStep` per screen**, in the order a person reaches them, named for what the person is
  doing. Assert what the person must see. Pass `{ a11y: true }` to audit the screen for WCAG 2.2 A
  and AA (see the `wcag-22-aa` skill).
- **Find elements by stable test ids** (`getByTestId`), role or label. Never by CSS class, layout or
  position.
- **A spec asserts exactly one outcome. No either-branch assertions.** Never branch on what an
  external supplier or any other uncontrolled answer returns (`if (found) {...} else {...}`, a
  try/catch that passes either way, a locator waiting for one of two results): the test is green
  whichever happens and proves nothing. Fix the answer in the fixtures and assert the one outcome it
  must give. Specs call no supplier; the platform walks each supplier path once against the real
  supplier when the build ends.
- **Seed through the app's API with `fixtureScope`**, which deletes exactly what the test created
  and fails the test if something is left behind. Never write to a database, never put records in
  the app's code, and never add a test-only route to the app (see "Seeding a deployed preview").
- **Seed bulk data with `fixtureScope.createMany(collection, items, { concurrency })` or in small
  bounded batches, never an unbounded `Promise.all` over `create`.** A hosted app's function runs at
  most 10 calls at once and the app's own traffic shares that budget; over it the API answers 503
  and the walk fails mid-seed. `createMany` keeps 4 in flight by default, and `create` retries a 429
  or 503 a few times with jittered backoff (honouring `Retry-After` up to the 5 s cap) before it throws. Seed the
  smallest set that proves the behaviour; that the app copes with a large list is the app's job
  (`limits.md`), not a reason to write thousands of fixtures. Fix a 503 here, in the spec, never by
  batching or throttling writes inside the app. On a deployed walk the journey `context` also holds the
  page's own API reads to 4 in flight and retries a 429 or 503 read with jittered backoff
  (`helpers/throttle.ts`), so a transient throttle does not fail a step. This stacks with the app
  client's own retry (bounded: at most 3 attempts each, so a read is tried at most 4 times by the
  harness, each of those up to 3 by the client). It must not hide a defect, so each screen's cost is
  reported (ledger line, `throttle-retries` annotation): a read still throttled after the retries
  fails the step, and a screen needing more than one retry or firing more than 8 reads gets a
  `request-budget` annotation. Treat that as the app's to fix: load the page's data through one
  aggregating endpoint (web-app-architecture `screen-data` pattern), not the spec's. Do not sign up or
  seed an account per journey: the demo session is the suite's one account and `fixtureScope`
  cleans what a test makes. The worker count is the walk's (`--workers=1`); if
  you run the suite yourself against a deployed app, keep `--workers` at 1 or 2.

## Seeding a deployed preview

A walk of a deployed preview seeds through the app's **real API**, as the demo identity, and nothing
else. Three rules:

- **Never a test-only route.** Do not add a `/test/seed`, `/api/test-fixtures` or any endpoint that
  exists to make test data, gated by an env flag or not. It ships to the deployed app, where anyone
  holding the demo account can write records that skip the real pipeline. A seed path that is
  local-only (`ALBITOR_LOCAL=1`) answers 404 on a preview and every walk fails at its first step. If
  a screen needs data, the spec makes it the way a person does: through the routes the product
  serves.
- **Seed once per preview, not once per walk.** A record that goes through a slow pipeline or a
  supplier (an upload that is processed, a document that is read) is made in a setup step and reused.
  Add a `seed.setup.ts` and a `seed` project in `playwright.config.ts` (`testMatch: /seed\.setup\.ts/`, `dependencies: ["setup"]`, `use: { storageState: DEMO_STORAGE_STATE }`, a long `timeout`), and add `"seed"` to `chromium`'s `dependencies`, and make the
  record with `seedOncePerPreview` from `helpers/preview-seed.ts`. It is keyed by the preview's
  base URL, so a new preview seeds afresh and every later walk of the same preview reads the saved
  ids (`readPreviewSeed`). Set `PREVIEW_SEED_VERSION` (a deploy id or commit) when a preview URL is
  redeployed in place, so the old seed is not reused. Two setup runs at once on one preview both
  seed; run one walk per preview at a time. Pass `stillValid` to re-check a saved record against the live app.
  Seeding through a real pipeline can take minutes: poll the app's own status route in the setup
  step with its own long timeout, and give the seed setup project one.
- **Specs that only read use the seeded record; specs that write use `fixtureScope`.** The seeded
  record is shared, so a spec never edits or deletes it.

```ts
// seed.setup.ts
import { test as setup } from "@playwright/test";
import { seedOncePerPreview } from "./helpers/preview-seed";

setup("seed the preview", async ({ request, baseURL }) => {
	setup.setTimeout(300_000);
	await seedOncePerPreview(baseURL!, "sample", async () => {
		// Drive the app's own API, with the demo session `request` already carries.
		const created = await request.post("/api/items", { data: { name: "sample" } });
		// ...wait for the app's own "ready" state, then return the ids the specs need.
		return { id: (await created.json()).id as string };
	});
});
```

### Media fixtures: short, real speech

A journey that uploads audio or video needs a fixture a speech service can read. A generated tone,
silence or a file of zeros makes the supplier answer with an empty transcript, and the app then
fails with no error to explain it.

- Keep it short: **a few seconds** (3 to 8) of a person, or a speech synthesiser, saying a plain
  sentence in the language the journey uses. Short keeps the repository small, the upload fast and
  the supplier bill low.
- Produce it, do not download it (no licence to check). Speak or synthesise one sentence, then
  trim and convert to a small mono file, for example:
  `espeak-ng -v en-gb -w raw.wav "The quarterly report is ready for review."` (or `say -o raw.aiff
  "..."` on macOS), then `ffmpeg -i raw.wav -ac 1 -ar 16000 -t 8 web/tests/fixtures/media/sample.wav`
  (use `-c:a aac sample.m4a` or `-c:v libx264` with a still image for another format). Play it back
  before committing it.
- Commit it under `web/tests/fixtures/media/` and assert on words the sentence contains. Do not
  assert exact transcription of proper nouns or rare words: a speech service may spell them
  differently.

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
  labelled "Password" and a button named "Sign in". A hosted sign-in (redirect to another host, Cognito managed login, branded from the app's look; the classic UI works too) is signed in there instead. An app
  with no sign-in (no password, no `services.auth`) runs signed out; one with a way in but no recognisable
  screen fails the setup. Specs that need a
  session skip themselves with `test.skip(!hasDemoSession(), ...)` in a `beforeEach`.

## Cross-account specs ("refuses another account's ...")

A spec proving one account cannot read or change another's records needs a second person. Use
`secondIdentityHeaders(request)` from `helpers/local-auth.ts`, never a spec written on the local
issuer alone: that skips on every deployed run and the journey is handed over unverified.

- **Local:** bob's bearer from the local issuer (`appAuthHeaders` is alice's).
- **Deployed:** the platform's seeded verifier-b. Its sign-in saves a browser session; the helper
  reads the ID token out of that session cookie and sends it as `Authorization: Bearer`, which wins
  over the primary's cookie on the same `request`.
- **Neither:** the test skips with `no second signed-in identity: this run has neither the local
  issuer (bob) nor a verifier-b session`. A walk reads that as unverified, not as a pass.

```ts
// @journey keep-claims-private
import { secondIdentityHeaders } from "../helpers/local-auth";
import { expect, test } from "../helpers/seed";

test("refuses another account's claim", async ({ request, fixtureScope, appAuthHeaders }) => {
  const other = await secondIdentityHeaders(request); // skips when there is no second identity
  const claim = await fixtureScope.create<{ id: string }>("/api/claims", {
    title: fixtureScope.uniqueName("claim"),
  });
  // The creator still reads it; the second account is refused (404 or 403, whichever the API gives).
  expect((await request.get(`/api/claims/${claim.id}`, { headers: appAuthHeaders })).status()).toBe(200);
  expect((await request.get(`/api/claims/${claim.id}`, { headers: other })).status()).toBe(404);
});
```

Assert the one status the API gives; do not accept either of two. For a browser check as the second
person, load `VERIFIER_B_STORAGE_STATE` into a second context (`hasVerifierBSession()` guards it).

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
