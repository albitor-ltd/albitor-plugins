# Changelog

All notable changes to the Albitor plugins catalogue. Plugin code changes are recorded in each plugin's own repository; this log tracks catalogue membership and the pinned versions.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Changed

- `font-clear`, `font-console`, `font-editorial`, `font-friendly`, `font-futuristic`,
  `font-geometric`, `font-high-legibility` and `font-technical`, each `0.1.1`
  (albitor-ltd/albitor#4852; tags follow the merge). The seeded `fonts.css` header comment no
  longer spells the CSP directive; it says the page's CSP only allows fonts from the app's own
  origin. The font gate fixture carries the same wording.

- `kit-shadcn` `0.10.0`, `kit-carbon` `0.9.0` and `kit-gds` `0.9.0` (albitor-ltd/albitor#4846; tags
  follow the merge). Per-component test fixtures, so the end-of-build prune (#4842) neither removes
  a fixture the kit's tests need nor keeps a component it could shed. The one shared
  `Showcase.svelte` (which mounted every component, so `knip --production` listed it as unused and
  a removed Dialog broke `kit.test.ts`) is gone: each kit ships a `__fixtures__/` folder with a
  small fixture per piece (`ShellFixture`, `FieldFixture`, `TableFixture`, `DialogFixture` and the
  kit's others) and the piece's own test (`field.test.ts`, `table.test.ts` and so on; `kit.test.ts`
  keeps the shell and, where the kit has one, the toaster), with the same assertions as before.
  `ContractProbe`, `SignInHarness` and gds's `RootClassesProbe` moved into `__fixtures__/` too.
  Each kit seeds `web/knip.json` (`ignore: src/**/__fixtures__/**`), so the web-svelte prune command
  never lists a fixture, and an app that never uses a piece deletes the piece, its fixture and its
  test as one unit. The skill's page examples split the same way: `pages/__fixtures__/` holds the
  frame (`PagesShowcase.svelte`, kept under that name because the platform's kit certification looks
  for it), a fixture per page template and shared helpers, with `listpage.test.ts`,
  `detailpage.test.ts`, `formpage.test.ts`, `dashboardpage.test.ts` and `settingspage.test.ts`
  beside `pages.test.ts` (now the sign-in page). `scripts/check-kit-fixtures.sh` enforces the new
  fixture locations.
- `web-svelte` `0.8.0` and `api-go` `0.9.0` (albitor-ltd/albitor#4842; tags follow the merge). Each
  declares `contributes.unusedCode`, the deterministic tool the end-of-build prune step runs to
  remove code nothing reaches, and the globs it never touches: `knip` pinned at `6.44.0` in
  production mode for the web tree (files, exports and types only; never dependencies), and
  `deadcode` pinned at `v0.48.0` with `-test` for the Go tree. Needs a platform that reads the key
  (the parser rejects an unknown `contributes` key), so land the platform change first.
- `web-svelte` `0.7.3` and `api-go` `0.8.1` (albitor-ltd/albitor#4820; tags follow the merge). The
  seeded `web/.gitignore` and `api/.gitignore` ignore test and coverage output (`test-results/`,
  `playwright-report/`, an app-root `/coverage/`, and for Go `*.coverprofile`; `*.out` was already ignored), so a
  test run's `.last-run.json` is never committed and handed over.
- `test-playwright` `0.6.6` (albitor-ltd/albitor#4810; tag follows the merge). Cross-account specs run
  on a deployed walk: `secondIdentityHeaders(request)` (`helpers/local-auth.ts`) gives bob's bearer
  from the local issuer locally and verifier-b's ID token, read from its saved session, on a deployed
  run, and skips with "no second signed-in identity ..." when neither exists. Pure helpers
  `bearerFromStorageState` and `resolveSecondIdentity` in `helpers/second-identity.ts`, with unit
  tests covering both paths. The skill gains a cross-account section and example spec. No
  `web-svelte` copy of these helpers exists.
- `test-playwright` `0.6.5` and `web-svelte` `0.7.2` (albitor-ltd/albitor#4779; tags follow the
  merge). Hosted request budget. The journey `context` of a deployed walk now keeps the page's own
  same-origin API reads inside the app's throttle (`helpers/throttle.ts`, with unit tests): at most
  four in flight, and a 429 or 503 read retried up to three times with full-jitter backoff,
  honouring `Retry-After` up to a 5 second cap; writes and streams are never touched. These retries
  stack, bounded, with the API client's own (up to 3 attempts each). They are reported, not hidden:
  per screen the ledger line carries `api_reads`, `throttle_retries`, `throttled` and `peak_queued`,
  the test gets `throttle-retries` and `request-budget` annotations, a read still throttled after
  the retries fails the step, and a screen needing more than one retry or firing more than 8 reads
  raises the `request-budget` finding. The seeded
  API client's `MAX_IN_FLIGHT` drops from 6 to 4 to sit well under the hosted function's 10
  concurrent executions. The `test-playwright` skill says a journey neither signs up nor seeds an
  account of its own.
- `test-playwright` `0.6.4` and `web-svelte` `0.7.1` (albitor-ltd/albitor#4779; tags follow the
  merge). The journey helper's `judgeScreen` now fails a step when the screen shows developer source
  text: a visible line starting `//` or `/*`, or containing `<!--`, outside `<pre>`, `<code>`,
  `<textarea>`, inputs and contenteditable regions, URLs excluded (pure rule `sourceTextProblem` in
  `helpers/screen-rules.ts`, with unit tests). A delivered app once rendered 18 `//` comment lines
  above its header on every signed-in screen and no walk noticed. The `web-svelte` skill says a
  component's file-level note goes in an HTML comment or inside `<script>`; no seeded or example
  component has a non-tag first line. The check reads rendered text (`innerText`), lives in
  `helpers/source-text.ts` with a `source-text.spec.ts` browser self-test, rethrows anything but a
  mid-navigation error, and skips any element carrying `data-allow-source-text`.

### Added

- `api-go` `0.8.0`, `auth-none` `0.5.0`, `auth-cognito` `0.10.0`, `auth-hosted` `0.5.0` and
  `data-dynamodb` `0.3.1` (albitor-ltd/albitor#4779; tags follow the merge). The kit's two-owner
  contract test. `api-go` seeds `internal/ownedscope` (stdlib only): a table-driven helper that reads
  the app's `openapi.yaml` (`x-data-model` ownership and each route's `x-entity`) and, for every
  route of a per-user entity, makes a record as one identity and asserts a second identity cannot
  read, update, delete or list it (a 404 or 403, and the list excludes it) and that a create with
  the same id leaves the first record intact. App-registered factories supply the bodies and ids;
  a route may be waived only with a written reason; a per-user route with neither a row nor a
  waiver fails the test. With `OWNED_DATA_SCOPE_REPORT` set it writes
  `{"operations":[{"method","path","outcome":"refused"|"leaked"|"waived","reason"}]}`. Each auth
  plugin seeds `internal/identity/owned_data_scope_test.go` (`TestOwnedDataScope`), which runs the helper
  against `Routes` with two injected identities; the skills tell the builder to keep it, register a
  factory for every per-user entity and never weaken it. `data-dynamodb` only gains skill text.
  The required set is the platform's own rule (`perUserOperations` in the albitor runner), copied and
  held to it by fixtures (`ownedscope/testdata/contracts`): for each collection with a create, the
  create, the list and the routes under one record (actions included), except a collection whose
  create names an entity the data model declares `shared`; the `servers[0]` base path is applied;
  admin detection matches (case-insensitive `admin`, scopes, path-item keys); the helper reads
  `openapi.yaml` with a stdlib reader that fails closed on YAML it cannot read. Admin-only operations
  are probed with a non-admin identity and must be answered 403; waivers are limited to reads (`GET`),
  with a specific reason of three or more words used on at most two operations, at most a quarter
  (minimum one) of the required operations; the report is merged into the package's file and
  rewritten atomically. The helper's own tests are named `TestScopeHelper...` and set the report
  variable themselves, so the platform's `-run TestOwnedDataScope` never runs or reads them. `auth-cognito`'s example contract marks the People admin routes `x-admin-only: true` and its examples add `TestOwnedDataScopePeopleAdminRoutes` (a non-admin is answered 403); the seeded identity test leaves `/api/auth/` to it through `Config.Elsewhere`. The skill
  says what the in-process test does not prove.
  `Config.Only` limits a package to its own path prefixes (the People admin test uses `/api/auth/`); waivers are refused when `Only` or `Elsewhere` is set, since the platform counts them across packages; a YAML anchor, alias or tag on a key, or an escape the reader does not know, fails closed.

- `font-high-legibility`, `font-technical`, `font-friendly`, `font-futuristic`, `font-geometric`,
  `font-editorial` and `font-console`, each `0.1.0` (albitor-ltd/albitor#4685; tags follow the
  merge). The seven remaining ADR-0107 `typeface@1` plugins, each self-hosting its OFL-1.1 woff2
  files with their licence text, a same-origin `fonts.css`, `preview-tokens.json` and a skill.
  Landed as one fan-in because each failed `just ci` alone unregistered.

- `font-clear` `0.1.0` (albitor-ltd/albitor#4684; tag follows the merge). The first `typeface@1`
  plugin and the worked example of ADR-0107: Source Sans 3 `3.052R` (Adobe, OFL-1.1, Reserved Font
  Name "Source") as the full unmodified upstream variable woff2 with its `OFL.txt`, a same-origin
  `fonts.css` (`font-display: swap`, size-adjusted fallback), `preview-tokens.json` and a skill. New
  gate `scripts/validate-fonts.sh` (run by `just ci`, fixtures in `tests/fonts/`), and
  `validate-preview-tokens.sh` accepts the `font_heading`, `font_body`, `font_data` and `font_label`
  stacks for a typeface plugin.

- `test-playwright` `0.6.3` and `web-app-architecture` `0.2.1` (albitor-ltd/albitor#4643; tags follow
  the merge). The seeded `FixtureScope.create` retries a 429 or 503 (a throttled call never ran the
  function) with bounded, jittered backoff honouring `Retry-After` up to a 5 second cap, and the new
  `createMany` seeds with 4 in flight instead of an unbounded `Promise.all`, so a walk no longer
  fails mid-seed against a hosted app's 10-call concurrency cap. `web-app-architecture` `limits.md`
  and the `test-playwright` skill say so.

- `test-playwright` `0.6.2` and `auth-hosted` `0.4.5` (albitor-ltd/albitor#3820; tags follow the merge).
  The hosted sign-in helpers sign in on Cognito managed login v2 as well as the classic UI (submit
  `button[type="submit"]` or `input[name="signInSubmitButton"]`; a refusal is also recognised by
  `[role="alert"]`, the v2 error element being unconfirmed by the captures). `auth-hosted` docs now
  describe sign-in as managed login, branded from the app's look, on the app's Cognito prefix domain;
  a branding change can take 1-3 minutes to show.

- `auth-hosted` `0.4.4` and `auth-cognito` `0.9.2` (albitor-ltd/albitor#4592; tags follow the merge).
  Their seeded `lib/auth/session.test.ts` now advances the timers across the API client's backoff
  when it asserts a 503 or network failure. Pairs with `web-svelte` 0.7.0 or later, whose client
  retries throttled reads.

- `web-svelte` `0.7.0` (albitor-ltd/albitor#4592; tag follows the merge). The seeded API client
  retries a GET or HEAD answered 429 or 503 (or an unreachable API) up to three attempts with
  full-jitter backoff never sooner than `Retry-After` (a `Retry-After` over 5 seconds fails at once),
  never retries a write, and holds at most six requests in flight per tab, abandoning an attempt after 30 seconds (a network failure,
  retried for reads) so a hung call cannot hold a slot; the query layer no longer
  retries what the client already retried. The `data-online` skill now says a screen gets what it
  renders from one API call and never fans out per row, because a hosted app's API is capped at 10
  concurrent executions.

- `auth-cognito` `0.9.0` and `deploy-byoa-aws` `0.9.0` (albitor-ltd/albitor#4452 item 15; tags follow
  the merge). An optional sign-up email-domain allow-list, `services.auth.signup_domains`, off by
  default: `deploy-byoa-aws` hands it to the function as `AUTH_SIGNUP_DOMAINS`, and `auth-cognito`
  refuses `sign-up` for other domains with 403 `email_not_allowed` and a clear message, and reports
  `signup_domains` from `GET /api/auth/config`. On both lanes the list holds every token the API takes
  (`auth-cognito` now does what `auth-hosted` does), so it also refuses existing users and invitees on
  unlisted domains and a user who registered directly with the pool: choose it before launch or include
  every current user's domain. The platform-seeded `.invalid` accounts always pass.
  `auth-hosted` already enforces `SIGNUP_DOMAINS` on every ID token; its docs now point at the manifest
  setting, which the manifest compiler renders as that config key. Land this PR first: it is inert
  until albitor-ltd/albitor's matching PR lands and a manifest compiler release carries the field.

- `object-s3` `0.1.0` (albitor-ltd/albitor#4317; tag follows the merge). The `object-store@1` plugin
  the composer maps `file-storage` to: a skill with examples over `services.objects` (presigned direct
  upload and download, owner-scoped keys authorised before signing, size and type limits, deleting
  with the record, retention), served on the app origin at `/_objects/<name>/` (hosted lane only), Go handlers over api-go and a web client
  for web-svelte. Nothing is seeded. Not in the catalogue until its first tag.

- `kit-shadcn` `0.8.0` (albitor-ltd/albitor-plugins#165, albitor-ltd/albitor#4316; tag follows the
  merge). `Table` and the `ListPage` example take `narrow` (`scroll` default, `stacked`, `priority`)
  and `Column.priority`, so a list stays usable on a phone; the skill gives guidance for choosing
  between stacked rows and priority columns, both shown and tested in `PagesShowcase`.

- `kit-shadcn` `0.9.0`, `kit-carbon` `0.8.0`, `kit-gds` `0.8.0` (albitor-ltd/albitor-plugins#181 items
  4 to 6; tag follows the merge). `AppShell` takes an optional `account` snippet in every kit and has
  a compact phone navigation (kit-shadcn and kit-gds: one Menu button below 48rem; kit-carbon: the
  side nav starts closed below its breakpoint and closes on navigation). kit-gds renders GOV.UK
  Service navigation properly: one `ul`, the `govuk-service-navigation__toggle` with `aria-controls`,
  `data-module="govuk-service-navigation"` initialised by Frontend's script, account links in the
  end slot, outside the list. `patterns.md` now matches the template. The skills no longer say to start
  a page from `Showcase.svelte`, and `scripts/check-kit-fixtures.sh` (in `just ci`, and over every
  composed tree in `just compose`) fails when anything but a test references a fixture.

- `scripts/check-internal-references.sh` (albitor-ltd/albitor#4518, albitor-ltd/albitor#4505): the
  delivered-tree mirror now scans every skill's `examples/`, expands declared globs and directories,
  strips the composer's `{{albitor.slot.*}}` and `{{albitor.app_name}}` placeholders, and runs over
  every plugin (`just manifests`). No plugin files change for it.

### Changed

- `web-app-architecture` `0.2.0` and `api-go` `0.7.0` (albitor-ltd/albitor#4592; tags follow the
  merge). Text only. `web-app-architecture`: `limits` gains every hosted limit (HTTP concurrency
  10 and its 503, API 50 rps and burst 100, schedules, queues, DynamoDB, logs) and a mitigation
  column naming the pattern that answers each; the new `screen-data` pattern says a screen gets
  what it renders from one call (lists embed summaries, screen-shaped endpoints, `ids=` batch
  reads, never a detail fetch per row). `api-go`: a "Shape endpoints for the screen" section.

- `api-go` `0.6.2` and `object-s3` `0.1.2` (albitor-ltd/albitor#4520, albitor-ltd/albitor#4505; tags
  follow the merge). Text only: `api-go`'s secrets guidance no longer mentions the retired
  `ALBITOR_SECRETS_PREFIX` alias (read `ALBITOR_SECRETS_PARAMETER_PREFIX`), and the `object-s3`
  example test's fixture bucket is `uploads-bucket`.
- `auth-cognito` `0.9.1` (albitor-ltd/albitor#4518, albitor-ltd/albitor#4505; tag follows the merge).
  Text only: the delete-account example comment no longer cites an internal decision record. The
  `object-s3` `0.1.2` example also swaps its `REC#2026/17` key fixture for `REC#a/17`, so no `#` is
  followed by three digits.

- Catalogue repinned to `api-go/v0.6.2`, `object-s3/v0.1.2` and `auth-cognito/v0.9.1`, all at
  7ef1e9d (albitor-ltd/albitor#4505). No other pin changes.

- Catalogue repinned to `web-svelte/v0.7.0`, `auth-hosted/v0.4.4`, `auth-cognito/v0.9.2`,
  `web-app-architecture/v0.2.0` and `api-go/v0.7.0` (albitor-ltd/albitor#4592). No other pin
  changes.

- Catalogue repinned to `test-playwright/v0.6.2` and `auth-hosted/v0.4.5` (albitor-ltd/albitor#3820).
  No other pin changes.

### Fixed

- `kit-shadcn` `0.9.1`, `kit-carbon` `0.8.1`, `kit-gds` `0.8.1` (albitor-ltd/albitor#4495; tags follow
  the merge). The seeded kit files open with "A starting point, then yours: ..." instead of "Seeded by
  Albitor (kit-...), then yours: ...", and their comments no longer name a plugin, a contract version
  or `auth@1`: a delivered app never names the platform or a plugin.
- `api-go` `0.6.1` (albitor-ltd/albitor-plugins#168; tag follows the merge). A supplier-client skill
  example (`examples/api/internal/supplier/`) reads the supplier's base URL from config with no
  default (`SUPPLIER_BASE_URL is not set` when empty) and shows honest fixtures: the supplier's
  documented response only, no invented fields, a missing documented field an error.
- `kit-shadcn` `0.7.1` (albitor-ltd/albitor#4313 item 17; tag follows the merge). `AppShell` centres its
  `main` content area (`margin-inline: auto`); above 72rem it sat flush left with the gap on the right.

### Changed

- `look-consumer` `0.1.3` (albitor-ltd/albitor-plugins#189, albitor-ltd/albitor#4496; tag follows the
  merge). Warm white page `#faf8f5` (was cream `#fff7ee`) and white cards `#ffffff` (was peach
  `#ffe8d1`); decorative border `#f2c9a0` to `#e8dccd`. The control border `#a97a50` is unchanged and
  still clears 3:1 on both. Every look-engine check passes in light and derived dark.

- `kit-carbon` `0.7.0`, `kit-shadcn` `0.7.0` and `kit-gds` `0.7.0` (albitor-ltd/albitor#4300; tags
  follow the merge). The page templates (`pages/*`, `PagesShowcase.svelte`, `pages.test.ts`) leave
  the seeded files and become worked examples in each kit's skill (`examples/web/src/lib/kit/pages/`,
  banners and skill pointers dropped); the components, the contract and every test stay seeded.
  `kit-gds` gains the page examples it lacked, built from its own GOV.UK components (summary list,
  error summary, notification banner), so it now passes `kit-pages-examples-are-sound`. Each kit's
  skill gains a page-templates section saying to copy from `examples/`.
- `deploy-hosted` `0.6.0` (albitor-ltd/albitor#4300, albitor-ltd/albitor-plugins#144; tag follows the
  merge). The gateway client is a skill example, not a seed: `api/internal/albitorsvc/**` no longer
  lands in a new app, and the package, renamed `gateway`, sits in
  `skills/deploy-hosted/examples/api/internal/gateway/` (with `queue.go` and its tests) for an app
  that declares `services.callable` or `queues` to copy in and edit. Only `ci.yml` is seeded. The
  example carries no platform name beyond the `ALBITOR_SERVICES_URL` and `ALBITOR_QUEUE_*`
  environment names. The `deploy-hosted` and `gateway-services` skills say when to copy it.

- `web-svelte` `0.6.0` (albitor-ltd/albitor#4300, albitor-plugins#142; tag follows the merge).
  Breaking seed change: the seeded data layer is online-only. `data-layer/index.ts` exports the
  online layer with no `{{albitor.data_layer}}` token, so the plugin no longer reads that platform
  token (a composer must still accept it until no lock names an older web-svelte). `lib/sync/**` and
  `data-layer/offline.ts` move to `skills/data-offline/examples/web/src/lib/`, to copy in and swap
  `index.ts` to `./offline`; the skill says how. `scope.d/data-layer.ts`, the shell, slot hosts,
  service worker and toolchain stay seeded, and the seeded tests import only seeded code. The
  "Starting point" and "Seeded by Albitor" banners and the pointers back to the plugin are gone.
- `auth-cognito` `0.7.0`, `auth-hosted` `0.4.0`, `auth-none` `0.4.0` and `api-go` `0.5.0`
  (albitor-ltd/albitor#4300, albitor-plugins#141; tags follow the merge). Breaking seed changes: the
  identity core stays seeded (`identity.go`, verify, token, the cookies and refresh, the hardening
  tests, `session.ts`, `guard.ts`, the slot fills and `provider.ts`) and is plain app code, with no
  banners or pointers back to the plugin. The sign-in calls, People admin, `client.ts`, the
  `(auth)` screens, the sign-in journey and the `/api/me` contract variant move to
  `skills/<plugin>/examples/`, to copy and alter; the skills say how, and the one join is a line in
  `registerIdentityRoutes`. `auth-cognito`'s guard no longer redirects the self-service screens in an
  invite-only app (that guard is an example). `api-go` always seeds `api/openapi.yaml`
  (the `when: auth-none` is gone) and its line 1 no longer names a plugin. `scripts/verify-auth.sh`
  now checks the seeded API alone and with the examples copied in. The `api-go` skill's queue
  guidance now names the `gateway` package, which the app first copies from the `deploy-hosted`
  skill's gateway example (see `deploy-hosted` `0.6.0`).

- `data-dynamodb` `0.3.0` and `api-go` `0.4.0` (albitor-ltd/albitor#4300; tags follow the merge).
  The seeded store, shell, router, routes, apikit and README are plain app code: the "Starting
  point" banners and every pointer back to the plugin or its skill are gone, and behaviour is
  unchanged. Both SKILL.md files list the seeded files and gain worked examples (a second table
  for `data-dynamodb`, a widget route and handlers for `api-go`).

- `deploy-hosted` `0.5.1` (albitor-ltd/albitor#4285, ADR-0104; tag `deploy-hosted/v0.5.1` follows
  the merge). `renders` gains `queues`: the hosted target now renders a declared queue, so a manifest
  declaring one no longer fails with `manifest_unrendered`. The `gateway-services` skill no longer
  says queues wait for a deploy target that renders them. Merge only after the cell has applied
  albitor-hosted-cells#59 and the dev channel pins module `v1.6.1`.

- `deploy-hosted` `0.5.0` and `api-go` `0.3.3` (albitor-ltd/albitor#4285, ADR-0104; tags
  `deploy-hosted/v0.5.0` and `api-go/v0.3.3` follow the merge). The `api-go` skill teaches the
  `POST /events` dispatch for schedules and queues (a schedule body has `schedule`, a queue's has
  `Records`), the idempotency rule, that `/events` is no longer public, and that the platform owns
  the local schedule tick so an app never runs its own loop or plans user-awaited work on a
  schedule. The seeded `albitorsvc` gains the queue half (`queue.go`): `OpenQueues`, `Send` and
  `SendBatch` to `ALBITOR_QUEUE_<NAME>_URL` signed with the function role, the envelope's `attempt`
  (at most 50) and `deadline` (at most one hour) bound, `ParseEvent`, `Failures` and `WriteResult`
  for `ReportBatchItemFailures`, and the in-process local queue; still standard library only. The
  `gateway-services` skill says to poll an async job by delayed re-enqueue, not on a schedule.
  `deploy-hosted` does not yet render `queues`: a manifest declaring one still fails early with
  `manifest_unrendered`.

- `deploy-hosted` `0.4.1` (albitor-ltd/albitor#4268, albitor#4273; tagged `deploy-hosted/v0.4.1`). The
  `gateway-services` request-shapes reference states the new 5,500,000-byte body ceiling (was 1 MiB)
  and documents Bedrock Converse media by `uploadHandle`: an image, document or video `source` is
  exactly one of `bytes` or `uploadHandle` (a video by handle only), obtained through
  `CreateUploadSlot` on `bedrock`, with an exact JSON block per media type and the `400` field names
  for both/neither and video-as-bytes. The Textract sync note and the Bedrock inline caps are
  reworded to the new ceiling (about 4 MB raw inline). Docs only; the seeded `albitorsvc` code is unchanged (its
  `Call` takes any request value).

- `deploy-hosted` `0.4.0` (albitor-ltd/albitor#4268; tagged `deploy-hosted/v0.4.0`). The `gateway-services`
  skill gains `references/request-shapes.md`: for every operation the gateway serves (Translate,
  Transcribe, Comprehend, Polly, Textract, Bedrock, Location) the fields allowed and required, what
  the gateway sets itself, and a minimal valid body, with the rule stated plainly that unknown fields
  are refused and AWS-SDK-only fields (Transcribe's `TranscriptionJobName`, `MediaFormat`) are never
  sent. The skill's idempotency step is corrected: the key is for `StartTranscriptionJob`, Textract's
  starts and Polly only, not Bedrock. The seeded `albitorsvc.Error` gains `Field` and `Detail`, read
  from the gateway's `field` and `detail` on a 400 (albitor#4271; empty until that gateway ships) and shown in `Error()`
  (`albitorsvc: 400 bad_request (field MediaFormat: unknown field)`); absent from an older gateway,
  the text is unchanged.
- `test-playwright` `0.5.2` signs in a second seeded identity on a deployed walk
  (albitor-ltd/albitor#4258, O29; untagged until released). The journey credentials file may hold
  `verifier_b` (`verifier-b@albitor-verify.invalid`); `auth.setup.ts` signs it in once after the demo
  account and saves `verifier-b.json` (`VERIFIER_B_STORAGE_STATE`, `hasVerifierBSession()`), so
  "another account cannot ..." specs run on a deployed app instead of skipping. It never fails the
  suite: a failed second sign-in (a pool not seeded with it) leaves no session, logs why and
  annotates `no-second-identity`, and only the specs that need a second person skip. New helpers
  `readVerifierB` and `saveSecondSession`, with unit tests.
- `auth-cognito` `0.6.0` is invite-only by default (albitor-ltd/albitor#4258, P2; untagged until
  released). It no longer hard-codes `sign_up: open` in its manifest (the plan supplies it, default
  `invite_only`). The API gains public `GET /api/auth/config`, `POST /api/auth/change-password` (both
  modes) and, in `invite_only` only, an administrator People admin (list, add with a generated
  temporary password shown once, disable, enable, reset) signed with a standard-library SigV4 client
  and guarded by the dedicated user-admin group (`AUTH_USER_ADMIN_GROUP`; empty fails closed); in `invite_only` sign-up, confirm, resend,
  forgot and reset answer 404. `AUTH_SIGN_UP` unset means `open`, so apps already deployed are
  unchanged. The web gains change-password and People screens and hides sign-up and reset when
  invite-only. Needs `deploy-byoa-aws` to hand `AUTH_SIGN_UP`/`AUTH_ADMIN_GROUP`/`AUTH_USER_ADMIN_GROUP` and the single-pool
  admin policy. `plugins/auth` skill `cognito-auth` and `auth-hosted` are docs only (no version change):
  the "no admin action on the API's role" rule is mode-aware, and hosted keeps self sign-up.
  `scripts/check-openapi-variants.sh` now also strips `/api/auth/*` blocks from the auth-cognito variant.
- `deploy-byoa-aws` `0.8.0` is invite-only without email (albitor-ltd/albitor#4258; untagged until
  released). With `services.auth.sign_up: invite_only` the user pool auto-verifies nothing and
  recovers accounts by an administrator only, so Cognito sends no verification, invitation or
  reset email (an in-place pool update, never a replacement; `open` is unchanged). The `auth`
  module's `env` gains `AUTH_SIGN_UP` and `AUTH_ADMIN_GROUP` and an output `user_pool_arn`; the
  `api` module gains `user_pool_arn` and `user_admin_enabled`, and in invite-only attaches one
  `user_admin` policy of exactly eight `cognito-idp` calls on that one pool (none in `open`). The seed
  creates the owner (`ALBITOR_OWNER_EMAIL`, `ALBITOR_OWNER_TEMP_PASSWORD`; `SUPPRESS`, a temporary
  password, the admin group; an existing owner is never touched) and `verifier-b@albitor-verify.invalid`
  (`ALBITOR_VERIFIER_B_ACCOUNT_PASSWORD`). `owner_status` is a new, additive key on the seed outputs
  and the deploy receipt; `seed_status` is unchanged. New managed workflow `recover-owner.yml`
  (manual dispatch; sets the temporary password Albitor minted and re-enables the owner), and the
  handover `docs/inviting-people.md` is rewritten for invite-only. The root template's five module
  `?ref=` strings point at `deploy-byoa-aws/v0.8.0`, a tag created after this change merges, as
  before. People administration is gated on a dedicated `user-admins` group (`AUTH_USER_ADMIN_GROUP`;
  created only in invite-only, the owner alone is seeded into it); `recover-owner.yml` refuses a
  temporary password written more than an hour ago; `invite.yml` refuses in invite-only. Platform-side work (the app's People screen, the plan's mode choice, the gate changes) is
  tracked in albitor-ltd/albitor#4258.
- `api-go` `0.3.2` teaches the schedule route (albitor-ltd/albitor#4247, stream #4248; untagged until
  released): the skill, the template README and `routes.go` say a schedule runs this binary behind
  the Lambda Web Adapter in pass-through mode, so each fire is `POST /events` with
  `{"schedule": "<name>"}`, and what the handler must do with it. Needs the hosted-app module that
  attaches the adapter to scheduled functions.
- Seeds boot as shipped (albitor-ltd/albitor#4228; patch releases `test-playwright` `0.5.1`, `api-go`
  `0.3.1`, `auth-cognito` `0.5.1`, `auth-hosted` `0.3.1`; untagged until released):
  the seeded `auth.setup.ts` signs in at api-go's local issuer (alice's token as the session cookie) in a local run, since
  the auth plugins answer 501 on `/api/auth/sign-in` there; the deployed path is unchanged.
  `test-playwright`'s Playwright config installs `web/` dependencies (`npm ci --include=dev`, then
  `npm install --include=dev`) when `node_modules` is absent before `npm run local`, which exited
  127 on a fresh checkout. `auth-cognito` and `auth-hosted` now seed `api/openapi.yaml` with the
  `GET /api/me` entry their sign-in probe depends on; `api-go` seeds its base contract only for
  `auth-none` (`scripts/check-openapi-variants.sh` keeps the copies in step).
- `deploy-hosted` `0.3.1`: the `albitorsvc` package comment no longer names the platform, so the
  seeded client passes the handover check (albitor-ltd/albitor#4216).
- Starting points, not managed files (albitor-ltd/albitor#4216, ADR-0103; minor releases of `api-go`
  `0.3.0`, `auth-cognito` `0.5.0`, `auth-hosted` `0.3.0`, `auth-none` `0.3.0`, `data-dynamodb`
  `0.2.0`, `deploy-byoa-aws` `0.7.0`, `deploy-hosted` `0.3.0`, `kit-carbon` `0.6.0`, `kit-gds`
  `0.6.0`, `kit-shadcn` `0.6.0`, `test-playwright` `0.5.0`, `web-svelte` `0.5.0`; untagged until
  released): every file a plugin contributes is seeded once and is the app's from then on, so every
  `managed` entry in the manifests is now `example`. Three kinds of entry keep their old mode only
  because the platform's parser or kit certification still requires it: the four slot templates
  (`router.go`, `app.html`, `nav.generated.ts`, `providers.generated.ts`, recorded as `slot-template`
  and rendered once), the workflows (`deploy-hosted` `ci.yml`, the four `deploy-byoa-aws` workflows,
  `test-playwright` `journeys.yml`, which the parser admits in `managed` mode only) and the kits'
  contrast test, `kit.test.ts`, `dialog.test.ts` and (`kit-shadcn`) `tokens.css`. Their headers and the skills and READMEs no longer
  claim a file is managed, generated or rewritten on compose or upgrade, or tell the builder to edit
  `albitor.yaml` instead of the file.
- The skeletons pass the floor as shipped. `deploy-hosted`'s `albitorsvc` client closes the response
  body without dropping its error (the `lint-clean/lint` finding); the delivered files no longer
  carry names the `handover-clean/no-internal-references` gate flags. Plugin-owned names changed:
  `ALBITOR_API_ORIGIN` is `APP_API_ORIGIN`, `ALBITOR_SIGN_IN_SCREEN` is `SIGN_IN_SCREEN`,
  `ALBITOR_SESSION_COOKIE_PATTERN` is `SESSION_COOKIE_PATTERN`, `ALBITOR_TESTS_OUT` is
  `TESTS_OUT_DIR`, `ALBITOR_TESTS_AUTH_DIR` is `TESTS_AUTH_DIR`, `ALBITOR_DYNAMODB_ENDPOINT` is
  `LOCAL_DYNAMODB_ENDPOINT`; the offline layer's IndexedDB prefix and lock name are `app-sync`; test
  keys `inv#001` are `inv#a1`; the contrast tests name their colour literals once. Still named by the
  platform and so still flagged until the platform's gate exempts them: the `albitorsvc` package and
  `ALBITOR_SERVICES_URL` (`deploy-hosted`), the `albitor-kit` cascade layer (`kit-shadcn`) and the
  `albitor:home-stub` marker (`web-svelte`).

### Fixed

- `api-go` `0.2.1` and `test-playwright` `0.4.2` (albitor-ltd/albitor#4175, stream #3944; `api-go`
  `0.2.1` also carries the #4173 change below; untagged until released): the `api-go`
  skill forbids a nil slice for any array field (build with `make`, clone with `make` and `copy`,
  never `append([]T(nil), ...)`), after a build shipped `defects: null` against an `openapi.yaml`
  array; the `test-playwright` skill forbids either-branch assertions, so a spec asserts exactly
  one outcome.

- `test-playwright` `0.4.1` (albitor-ltd/albitor#4177, stream #3944): the suite's signed-in session is
  written to a private (0700) directory under the OS temp dir, not to `web/tests/.auth/` in the tree,
  where a secrets scan walking the tree flagged it. `ALBITOR_TESTS_AUTH_DIR` overrides the directory;
  `helpers/storage-state.ts` gains `ensureAuthDir()`, which `auth.setup.ts` and the `auth` plugin's
  `auth.setup.ts.tmpl` call. Untagged until `test-playwright/v0.4.1` is cut; the catalogue pins the
  previous version meanwhile. `ensureAuthDir()` chmods 0700 only a directory it created or the default
  one; an existing override open to group or others is refused, never chmodded.
- `auth` `0.7.1` (albitor-ltd/albitor#4177): `auth.setup.ts.tmpl` imports `ensureAuthDir` from
  `helpers/storage-state.ts`, so it requires `test-playwright` >= 0.4.1. Release and pin the two
  together.

- `deploy-byoa-aws` and `api-go` (albitor#4173; `api-go` `0.2.1` ships it; unreleased, the release PR
  bumps `deploy-byoa-aws` to `0.6.1`): the API function can read the app's own SSM secrets. The `api` module takes `secrets_parameter_prefix` (empty grants nothing), grants
  `ssm:GetParameter` and `ssm:GetParameters` on it, sets `ALBITOR_SECRETS_PARAMETER_PREFIX` and outputs
  the role; the root stack derives the prefix from `secrets_segment`, which `deploy.yml` passes from
  `ALBITOR_APP_SECRETS_SEGMENT`. The `api-go` skill records the variable and the read convention.
  The catalogue still pins `deploy-byoa-aws` `0.6.0` and `api-go` `0.2.0` until released.

- `deploy-byoa-aws` `0.6.1` (albitor-ltd/albitor#4188, stream #3944): every file the plugin renders,
  and the modules the stack fetches, passes the platform's internal-references check, so no agent is
  asked to edit a managed file. The resource tag is `Template = "deploy-byoa-aws"` (was prefixed with
  the platform's name), the `main.tf` and seed-script headers and comments no longer cite the platform,
  an issue or an ADR, and the manifest path is joined from segments. No resource address changes
  (`module.site[0]` and the rest stay), so no `moved` block is needed; the tag value updates in place.
  `scripts/check-internal-references.sh` runs the check (a mirror of the platform's handover gate; that
  script is the source of truth) over the manifest's `contributes.files` and `modules/`, and
  `scripts/test-check-internal-references.sh` proves it fails on each kind of reference. Both run in
  `just deploy-byoa-aws`. Release `0.6.1` together with the #4173 change above; the version is bumped and
  the module refs point at `deploy-byoa-aws/v0.6.1`, and the catalogue still pins `0.6.0` until the tag
  exists.

### Added

- `deploy-byoa-aws` (albitor-ltd/albitor-plugins#105, albitor#4040 Delete app): a managed
  `destroy.yml`, dispatch-only with `environment` and `ref` inputs, under the same OIDC deploy role and
  `deploy-<env>` concurrency group as `deploy.yml`. It empties the site bucket's object versions, lifts
  table and user pool deletion protection, runs `terraform destroy`, and a job named exactly
  `destroy receipt` emits `::notice title=destroy-receipt::{environment, commit, state}`. Released as
  `0.6.0` (albitor-ltd/albitor-plugins#106 added the receipt); the version is bumped and the module
  refs point at `deploy-byoa-aws/v0.6.0`, and the catalogue pins `0.6.0` (tag `deploy-byoa-aws/v0.6.0`,
  commit `c4ca02d`).

- `deploy-hosted` `0.2.0` (albitor-ltd/albitor#4005, ADR-0099 section 3.1): `renders` gains
  `services.callable`; `api/internal/albitorsvc` (managed, standard library only), the thin SigV4
  client for the service gateway with typed 402/403/429 errors, jittered capped backoff and
  upload-slot helpers; the `gateway-services` skill (declare `services.callable`, sign-in or
  per-end-user limits by default, `402` as "feature unavailable"). `just ci` runs
  `scripts/verify-deploy-hosted.sh`. Untagged, so the catalogue still pins `0.1.2` until released.

- The `auth@2` interface, `roles()` / `seed(role)` / `signInAs(role)` (albitor-ltd/albitor#3937 part B,
  stream #3944). `test-playwright` `0.4.0` ships it (`helpers/auth-v2.ts`, `helpers/roles.ts`, the
  sign-in flows moved to `helpers/browser-signin.ts`) and `roles.spec.ts`, the per-role check: every
  declared role signs in as an account holding that group alone. `auth-none` `0.2.0`, `auth-cognito`
  `0.4.0` and `auth-hosted` `0.2.0` each ship `web/tests/auth/provider.ts`, the plugin's provider.
  `api-go` `0.2.0`: the local issuer seeds a role (`POST /api/_local/auth/seed`, identity
  `role-<group>`, that group only). `deploy-byoa-aws` `0.5.0`: the seed gives every
  `services.auth.groups` entry its own account (`<role>@albitor-role.invalid`, password
  `ALBITOR_ROLE_<ROLE>_ACCOUNT_PASSWORD`), root output `auth_groups`, and `seeded_roles` lists every
  role seeded; the demo account is not given the other groups. `web-svelte` `0.4.3`: the home page
  stub carries `<!-- albitor:home-stub -->`, which the foundation gate fails on until it is replaced.
  The manifests still `provide` `auth@1`: the socket registry accepts `auth@2` only once albitor
  adds it. Untagged, so the catalogue still pins the previous versions until released.
- `deploy-byoa-aws` `0.4.0` (albitor-ltd/albitor#3922): `services.auth.admin_group` (default the first
  group, must be one of `groups`); the deploy's seed adds the demo and verifier accounts to it
  (`cognito-idp:AdminAddUserToGroup`, idempotent) and the deploy receipt carries `seeded_roles`.
  `auth-cognito` `0.3.2` and `auth-hosted` `0.1.3`: SKILL.md says an app has one kind of account unless
  the plan declares roles, privileged routes gate on the declared `admin_group`, and nobody is asked
  to change groups. Untagged, so the catalogue still pins the previous versions until released.
- `scripts/check-runner-playwright.sh` and `runner-image.json` (albitor-ltd/albitor#3895): CI fails when
  `test-playwright`'s Playwright pin differs from the version the Albitor runner image bakes a browser
  for, so a dependency bump cannot leave offline spec runs and journey walks without a browser. Bump
  the image first, then the record and the plugin together. No plugin version change.

### Fixed

- Look contrast (albitor-ltd/albitor#3944). `look-startup` `0.1.2`: accent `#0a7f8f` to `#08737f`
  (4.45:1 to 5.24:1 on the `#f7f8fa` page, 5.57:1 on white), so links pass the rendered axe scan.
  `look-startup`, `look-business` and `look-consumer` `0.1.2`: the dark page is no longer left on the
  light accent. An unlayered override beats kit-shadcn's layered dark block, so SKILL.md now says to
  override tokens in both the light and dark selectors, or not at all, and names a dark accent that
  passes AA on `#09090b` (startup `#2fb5c6`, business `#5aa9f0`, consumer `#f06cd0`). Dark values are
  not authored in `preview-tokens.json` (the platform derives `dark_*` tokens and the allow-list
  refuses them). Untagged, so the catalogue still pins the previous versions until released.
- Fixes for findings the Albitor task-time floor gates raised on the seeded trunk (not a claim that
  every gate now passes). `auth-cognito` `0.3.1` and `auth-hosted` `0.1.2`: the
  line-1 "Seeded by" header on each seeded screen names only the plugin, so it matches the
  handover-clean exemption (the `@` in `component-kit@2` broke it). `data-dynamodb` `0.1.2`: the
  response body close is checked (errcheck), and the SigV4 test vectors are assembled at run time
  so the committed-secrets scan no longer flags them. `auth-cognito` `0.3.1`: a tagged switch in
  the identity test (staticcheck QF1002). `auth-hosted` `0.1.2`: the unused `verify` wrapper is
  removed. `web-svelte` `0.4.2`: the error-message test no longer uses `secret` and `token=` in
  its fixture. `auth-hosted` `0.1.2`: the sign-in page's sessionStorage key is `app.sign-in.at`
  (was a platform-named key; nothing else reads it). Tests assert the same behaviour.

- `auth-hosted` `0.1.1` (albitor-ltd/albitor#3870; `0.1.0` was already tagged, so this is a new
  version): `acceptIDToken` always accepts the platform-seeded domains `albitor-verify.invalid` and
  `albitor-demo.invalid` (a verified address is still required), whatever `SIGNUP_DOMAINS` says, so
  an app that sets it no longer locks out the demo and verifier accounts. The README records why no
  outsider can hold a verified address there. The skill tells agents not to set `SIGNUP_DOMAINS`
  unless the brief or a person asks for a domain restriction.

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

- Go 1.26 becomes 1.27, `api-go` `0.1.2`: the template `api/go.mod` is `go 1.27`, so every generated app
  builds on Go 1.27. This repository's CI `setup-go` jobs (go1.27.3), `tests/compose/go.mod` and the
  `compose-check.sh` workspace move with it. The other plugins' workflows read `api/go.mod` and follow. No
  golangci-lint is pinned here. Needs the `api-go/v0.1.2` tag before the catalogue can pin it, and the
  catalogue should pin it only after albitor's runner image is on Go 1.27.
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
