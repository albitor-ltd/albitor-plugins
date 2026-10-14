# web-svelte

The `web-framework@1` plugin for SvelteKit
([ADR-0090](https://github.com/albitor-ltd/albitor) section 2 and 5, albitor-ltd/albitor#3165). A
dedicated shell, composed into a new app's `web/` by the platform's composer.

## What it provides

- A SvelteKit static SPA build (`adapter-static`, fallback `index.html`, `ssr = false`).
- The slot host: a managed root layout that nests `web.providers`, runs `web.route-guard` before
  every navigation and hands the merged `web.nav` to the kit's `AppShell`.
- Error, 404 and offline pages (seeded, app-owned) and a service worker that opens the built app
  offline.
- A small API client (`src/lib/shell/api.ts`): app-relative paths to `/api`, JSON, bearer token source,
  `ApiError` and `NetworkError`.
- The web side of the local run contract: `npm run local` builds and serves `build/` with `/api`
  forwarded and `/healthz`; `npm run dev` forwards `/api` too.
- The `svelte-shell` skill: how to add a route, a page and a nav entry.

## Conventions the composer renders

| Slot | Convention |
|---|---|
| routes | App-owned SvelteKit files under `web/src/routes/`. No slot, no managed file to edit |
| `web.nav` | A route adds a sibling `nav.ts` exporting `nav = { label, order, group }`; the managed `nav.generated.ts` collects them with `import.meta.glob('/src/routes/**/nav.ts')` and merges plugin entries. A plugin entry has no target: its link is the label slug (`Sign in` is `/sign-in`) |
| `web.providers`, `web.route-guard` | A plugin ships a module in `web/src/lib/shell/scope.d/` exporting the identifier it declared; the managed `providers.generated.ts` resolves it |

## Dependency pins

`contributes.dependencies.npm` and the seeded `web/package.json` are exact pins, at least 14 days
old. `@sveltejs/kit` 3.0.1 (2026-10-06) and `@sveltejs/adapter-static` 4.0.0 (2026-10-01) are the
exception: no older release of their major exists. Transitives in the seeded lockfile are not
subject to the rule (current practice).

## What it needs from other plugins

- `component-kit@1`: a `web/src/lib/kit/AppShell.svelte` taking `nav: NavGroup[]`,
  `path: string` and `children: Snippet` (see the skill).
- `deploy-target@1`: only because this plugin contributes the npm entry for `repo.dependabot`.

## Layout

```text
.claude-plugin/plugin.json   schema-2 manifest
slot-templates.json          the generated slot files this plugin designates
slot-templates/              their bodies, with {{albitor.slot.*}} placeholders
templates/web/**             the files composed into the app, at their repository paths
skills/svelte-shell/         the skill
```

The slot-template layout (`slot-templates.json`) is provisional pending
albitor-ltd/albitor#3215. A route guard is a convenience, not access control: the API must check
the caller. The guard runs in `+layout.ts` `load` before providers mount, so there is no
session-state ordering guarantee, and tokens must not be persisted in `localStorage`. A plugin
`web.nav` entry must ship the route its label slugs to.

`slot-templates.json` is read by whoever builds the composer's `Plugin.SlotTemplates` from this
tree. Compose, build and test it with `scripts/compose-check.sh` (see the repository README).
