# Albitor plugins

The first-party plugin monorepo and marketplace catalogue for
[Albitor Ltd](https://github.com/albitor-ltd)'s
[Claude Code](https://code.claude.com/docs/en/plugins) plugins (albitor-ltd/albitor#3158,
ADR-0090 section 12).

## Install

```bash
claude plugin marketplace add albitor-ltd/albitor-plugins
claude plugin install gds@albitor-plugins
```

## Layout

```text
.claude-plugin/marketplace.json   generated catalogue - never hand-edit
plugins/<name>/                   one directory per first-party plugin
  .claude-plugin/plugin.json      the plugin manifest (name must equal <name>)
catalogue/metadata.json           catalogue-facing description, category, keywords, licence
catalogue/legacy-pins.json        old per-plugin repo and ref; history, used only by --mode legacy
scripts/build-catalogue.sh        generates marketplace.json from the tree
scripts/validate-plugins.sh       manifest validation (run by `just ci`)
scripts/check-runner-playwright.sh test-playwright's Playwright pin equals runner-image.json, the version the runner image bakes a browser for (run by `just ci`)
scripts/validate-preview-tokens.sh  every plugins/*/preview-tokens.json against the platform's contract
scripts/test-preview-tokens.sh    accept/refuse cases for that validator (tests/preview-tokens/)
scripts/compose-check.sh          real parser and composer over schema-2 plugins, then the web build and, with --journeys, the Playwright journeys
tests/compose/                    the Go test that script runs, with stub plugins
```

| Plugin | What it does |
| --- | --- |
| `gds` | GOV.UK Design System and WCAG 2.2 AA accessibility |
| `shadcn` | shadcn/ui on Radix UI and Tailwind CSS |
| `mui` | Material Design (SMUI on SvelteKit, `@mui/material` on React) |
| `carbon` | IBM Carbon Design System for enterprise interfaces |
| `auth` | Cognito-backed sign-up and login |
| `docs` | Diataxis, C4 and arc42 documentation skills |
| `web-svelte` | The `web-framework@1` socket: SvelteKit static SPA, shell, slot host, and both client data layers (online-only TanStack Query, or offline-first `@albitor/sync` outbox and sync), the platform's `data_layer` token selecting one (schema-2 manifest; monorepo-only, not in the legacy catalogue) |
| `api-go` | The `api-runtime@1` plugin: Go server skeleton, middleware and route slots, paging, local issuer |
| `test-playwright` | The `test-harness@1` plugin: Playwright journey harness, sign-in setup, fixtures, axe WCAG 2.2 A/AA audit, a read-only CI workflow |
| `data-dynamodb` | The `data-store@1` plugin: `Store` over one DynamoDB table, in-memory fake, conformance suite, `services.tables` fragment |
| `kit-shadcn` | The `component-kit@1` and `component-kit@2` plugin for Svelte: shadcn-style shell pieces on Bits UI, one token set, contrast and axe audits that ship with the app |
| `kit-carbon` | The `component-kit@1` and `component-kit@2` plugin for Svelte in the IBM Carbon style: Carbon's UI shell and components, its white and Gray 100 themes, contrast and axe audits that ship with the app; not themeable by a visual-language plugin |
| `kit-gds` | The `component-kit@1` and `component-kit@2` plugin for Svelte in GOV.UK Design System style: GOV.UK Frontend v6 markup, a text-only header (no Crown or logotype), the GOV.UK design-system and accessibility skills, contrast and axe audits that ship with the app; not themeable |
| `deploy-byoa-aws` | The `deploy-target@1` plugin for a customer's own AWS account: Terraform rendered from `albitor.yaml` (site, API, tables, alarms, DNS) from versioned modules, the CI and deploy workflows (OIDC), the stale state-lock guard |
| `deploy-hosted` | The `deploy-target@1` plugin for the hosted lane: no infrastructure in the repo, the platform compiles `albitor.yaml` and deploys; the managed CI workflow (manifest check, API and web build and test), and `albitorsvc`, the thin Go client for the service gateway (`services.callable`) |
| `auth-cognito` | The `auth@1` plugin for the default profile: Cognito sign-up and sign-in, the API's identity seam, a session module and route guard, sign-up, sign-in and reset screens on any `component-kit@2` kit, `services.auth` |
| `auth-hosted` | The `auth@1` plugin for the hosted lane: sign-in through the app's Cognito hosted page (authorization code with PKCE), the API's identity seam and callback, a session module and route guard, sign-out, account deletion and sign-up domain refusal screens with no password field, `services.auth` |
| `auth-none` | The `auth@1` plugin for an app with no sign-in: a stub identity seam with one fixed owner, no screens |
| `look-business` | The `visual-language@1` look for enterprise and line-of-business apps: dense, navy neutrals, 2px corners, no elevation, tabular figures; ships `preview-tokens.json` |
| `look-startup` | The `visual-language@1` look for SaaS and developer tools: near-neutral surfaces, one teal accent, 10px corners, hairline plus soft layered shadow; ships `preview-tokens.json` |
| `look-consumer` | The `visual-language@1` look for mobile-first personal and lifestyle apps: warm cream and berry, rounded type, 24px corners, 48px targets; ships `preview-tokens.json` |

`auth-cognito` and `auth-none` are the split of `auth`; `auth` stays as released (`auth/v0.7.0`). `auth-hosted` is a separate `auth@1` provider for the hosted lane (the pool comes from the hosted-app auth module and signs in on Cognito's hosted page); `auth-cognito` stays the plugin for an app's own account (BYOA), which signs in with a password through its own screens.

`client-online` and `client-offline` (the retired `client-data@1` socket) folded into `web-svelte`
0.2.0 (albitor-ltd/albitor ADR-0090 amendment of 2026-10-15); their `v0.1.0` tags stay for history.

The old `visual-*` packs are not migrated; they are retired (albitor-ltd/albitor#3176, #3530).
`mui` and `gds` are parked for the beta; `docs` stays a skill pack (ADR-0090 section 13).

## Tags and releasing a plugin

Each plugin is released independently with a tag `<name>/vX.Y.Z`, where `X.Y.Z` is the `version` in
`plugins/<name>/.claude-plugin/plugin.json` (for example `auth/v0.7.0`).

1. Change the plugin and bump `version` in its `plugin.json` in the same PR.
2. Merge to `main` once `just ci` passes, plus `just compose <albitor checkout>` for template, manifest or composer-contract changes.
3. Tag the merge commit: `git tag auth/v0.7.0 && git push origin auth/v0.7.0`.
4. In a follow-up PR, regenerate the catalogue (`scripts/build-catalogue.sh`, after `git fetch --tags`) so it
   pins the new tag and its commit. Only tag-then-regenerate keeps the catalogue from pointing at a tag that does
   not exist yet.

## The catalogue

`.claude-plugin/marketplace.json` is generated by `scripts/build-catalogue.sh` and `just ci` fails on drift. Two modes:

- `monorepo` (default, and what is committed): one subdirectory source per plugin, pinned to its
  release tag and to the commit that tag resolves to:

  ```json
  { "source": "git-subdir", "url": "albitor-ltd/albitor-plugins",
    "path": "plugins/auth", "ref": "auth/v0.7.0", "sha": "<40-character commit>" }
  ```

  A `git-subdir` source takes `url`, `path`, `ref` and an optional full 40-character `sha`, and
  checks out only the subdirectory (Claude Code plugin marketplace reference). The generator reads
  `sha` from the local tag, so a tag that moves makes `--check` fail and a missing tag fails
  generation. A plugin not yet in the committed catalogue and with no tag is left out (with a note)
  until its first tag, so a new plugin's pull request can pass `--check`; one already catalogued
  whose tag is missing is an error. Albitor's pin audit joins entries on
  repository and subdirectory.
- `legacy`: `github` sources pointing at the old `albitor-plugin-<name>` repositories and tags
  (`catalogue/legacy-pins.json`). Kept only so the old shape
  can still be generated; nothing is committed in it.

### Switch-over

Done: every first-party plugin is tagged `<name>/vX.Y.Z` from this repository, and the committed
catalogue is in `monorepo` mode. Remaining, outside this repository: archive the old
`albitor-plugin-*` repositories with a pointer README once no tenant source points at them (keep
their tags ingestible until then).

## The `component-kit@2` contract

A kit that follows the `component-kit@2` contract (ADR-0095 section 6) exports three standard shell components from
`web/src/lib/kit/index.ts`, with the same names and props in every kit. A plugin that renders
screens (an auth plugin, a page template) imports them from `../lib/kit` and never branches on which
kit is installed. The contract names no Style; each kit draws the pieces in its own idiom.

```ts
import { AppShell, Footer, SignInPage } from '../lib/kit';
import type { FooterLink } from '../lib/kit';
```

| Export | Props | Landmarks and behaviour |
| --- | --- | --- |
| `AppShell` | `nav: NavGroup[]`, `path: string`, `title?: string` (the app's name), `footer?: Snippet`, `children: Snippet` | Skip link first and pointed at the one `<main>`; one `<nav aria-label="Main">`; `aria-current="page"` on the current link and `"true"` inside its section; renders `footer` after `<main>`, outside it. web-svelte's root layout renders it. |
| `Footer` | `links?: FooterLink[]` (`{ href, label }`), `children?: Snippet` (free text) | One `<footer>` (the contentinfo landmark), never inside `<main>`; links in `<nav aria-label="Footer">`; labels and text rendered as text; renders nothing when given neither. Pass it through `AppShell`'s `footer`. |
| `SignInPage` | `title?: string`, `error?: string`, `busy?: boolean`, `resetHref?: string`, `signUpHref?: string`, `email?: string` (the email field's initial value), `onsubmit(email, password)` | Presentation only: the auth plugin decides what signing in means. One `<h1>`; labelled email (`autocomplete="username"`) and password (`autocomplete="current-password"`) fields; a failure is one focused alert carrying the `error` sentence (which never says which field was wrong); the button is disabled while `busy`. |

```svelte
{#snippet footer()}<Footer links={[{ href: '/privacy', label: 'Privacy' }]}>Example Ltd</Footer>{/snippet}
<AppShell {nav} {path} {title} {footer}>
  <SignInPage {error} {busy} resetHref="/reset" onsubmit={(email, password) => signIn(email, password)} />
</AppShell>
```

Each kit ships `contract.test.ts` and `ContractProbe.svelte` (managed): the exports exist, the
landmarks above hold, the sign-in page and the footer pass axe. A kit's other pieces (`Button`,
`Field`, `Table`, ...) stay kit-specific in props that match across kits where they can, but only
these three are the contract. **Upgrading an existing app.** The managed
tests import the kit's `index.ts`, which is the app's after the first compose, so the new tests fail
until the app adopts the exports: take the kit's new `Footer.svelte` and its changed `SignInPage.svelte` (kit-shadcn and
kit-carbon: `pages/SignInPage.svelte`, whose error-focus effect changed, so `signin.test.ts` fails
against the old one; kit-gds: a new file) from `templates/web/src/lib/kit/`. They are `example`
files, and what compose does with them on an upgrade is under review in albitor-ltd/albitor#3722, so
check your `web/src/lib/kit/` files after upgrading rather than relying on either behaviour. Also
add `Footer`, `FooterLink` and `SignInPage` to `index.ts`, and add the optional `footer` snippet to `AppShell.svelte`, rendered after `<main>`. **Declaration.** The platform registry accepts major 2 (albitor-ltd/albitor#3746), so each kit declares both
majors, `provides: ["component-kit@1", "component-kit@2"]`: a consumer's `requires` resolves against an exact major, and
web-svelte still requires `component-kit@1`. A consumer of the standard shell (`auth-cognito`) requires `component-kit@2`.

## Preview tokens

A design-system or visual-language pack may ship `preview-tokens.json` at its root: the small
token set the platform paints a browser preview with. The platform reads it strictly
(`governance.PreviewTokens` in albitor-ltd/albitor: unknown keys, an unread `version` or any value
outside the allow-list make it ignore the whole file), `just ci` runs `scripts/validate-preview-tokens.sh`
over every pack and fails on the same grounds. Required: `version` (1), `font_family`,
`heading_weight` (100-900, hundreds), `type_scale` (`small`, `body`, `heading`, `title`), `radius`,
`density` (`compact|comfortable|spacious`), `elevation` (`none|subtle|raised|layered`) and
`colours` (`text`, `text_secondary`, `border`, `surface`, `background`, `accent`, `on_accent`, hex).
Optional: `heading_font_family`, `border_width`, `min_target`, `focus_ring`, `heading_tracking`
(`0` or `-0.02em`), `tabular_figures`, `dark_target` (0 or 4.5-7) and the colours `control_border`,
`danger`, `success`, `warning`, `warning_alt`. An optional `roles` block names colours kept apart
from the action colour (all hex, all optional): `header` (`background` and `foreground`, which travel
together), `header_withdrawn` (the same, used only where the platform withdraws the buyer's brand
colour; a brand colour the buyer set paints the header instead, else `header`), `link`, `focus` and
`focus_ink` (text on a focus fill: with it the focus indicator is a `focus` fill with `focus_ink`
text and underline, without it a `focus` ring). Lengths are `0` or up to four decimals of `px`/`rem`;
the file is at most 4 KiB. Run `scripts/validate-preview-tokens.sh` before opening a pull request.

## Gate

There is no hosted CI (see #98), so `just ci` is the merge gate; run it before merging. Release
tags are not CI-certified on push, so run `just ci` on the commit before tagging. It does not run
`compose`, which needs a checkout of albitor-ltd/albitor and Node. Before merging a change to plugin
templates, manifests or the composer contract, also run `just compose <albitor checkout>` (`scripts/compose-check.sh --build`).

## CI

`just ci` runs: manifest validation (JSON parses, `name` equals the directory, semver
`version`, catalogue metadata present), preview-token validation (`scripts/validate-preview-tokens.sh`
and its refusal tests), catalogue reproducibility (`--check`), shellcheck and JSON lint, and the
plugin verifies (api-go, data-dynamodb, auth, deploy-byoa-aws).

Compose and certification run only via `just compose <albitor checkout>`: the real manifest parser
and composer from albitor-ltd/albitor over the schema-2 plugins, then `npm install`, check, test and
build of the composed `web/`. Add `--journeys` (`scripts/compose-check.sh --build --journeys
<albitor checkout>`) to run the `test-playwright` journeys in Chromium. It also runs
`tests/compose/kit_look_test.go`, the kit x look matrix (a themeable kit composes with every look,
one that is not is refused), and `tests/compose/certify_test.go`: the platform's certification
harness over every socket plugin and every certified profile (default, byoa-aws), with the deploy
target required to declare `renders`, the composed `albitor.yaml` checked against it, and
`terraform init -backend=false` plus `terraform validate` over each composed Terraform root, with no
cloud credential (albitor-ltd/albitor#3333). `tests/compose/policy_packs_test.go` composes the
default selection with each built-in policy pack mandated (Internal Tool clean; GDS refuses only a
visual-language provider; albitor-ltd/albitor#4156). Run it on the commit before tagging a release
(`<plugin>/v<semver>`).

## Third parties

Nothing here constrains anyone else. A plugin source may be a repository root, a subdirectory of a
repository, a git URL or an npm package, so third parties publish one repository per plugin or their
own monorepo. The platform runs the same conformance and certification harness at ingest that this
repository's gate.

## Air-gapped / offline use

Internet-connected installs use the command above. Air-gapped or regulated deployments are handled
per engagement: clone this repository, assemble a local marketplace (relative-path sources) or load
via `claude --plugin-dir plugins/<name>`, and sign as required.

## Licence

The catalogue (`marketplace.json`, scripts and docs) is MIT, see [`LICENCE`](LICENCE). Each plugin
carries its own licence in `plugins/<name>/LICENCE`; bundled standards content (for example
GOV.UK and WCAG material in `gds`) is redistributed under the Open Government Licence v3.0 and the
W3C Document Licence, see that plugin's `NOTICE`.
