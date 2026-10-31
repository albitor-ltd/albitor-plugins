---
name: svelte-shell
description: How to add a route, a page and a nav entry on the web-svelte shell, and what the shell owns. Use when a build touches web/src/routes, the nav, providers or the route guard of a SvelteKit app composed from the web-svelte plugin.
---

# The web-svelte shell

The app is a SvelteKit static single-page app under `web/`. Routes are SvelteKit's own file routing
and belong to the app, so **adding a page never edits a shell file**.

## Add a page

1. Create `web/src/routes/<name>/+page.svelte`. A route group in parentheses, such as
   `web/src/routes/(app)/reports/+page.svelte`, shares a layout and adds nothing to the URL.
2. Give it a `<svelte:head><title>` and one `<h1>`.
3. Load data with the API client, in the page or a `+page.ts`:

   ```ts
   import { get, post, ApiError, NetworkError } from '../../lib/shell/api';
   const items = await get<Item[]>('/items');
   ```

   Paths are app-relative and always go to this app's `/api`. Catch `NetworkError` (check `.offline`)
   and `ApiError` (`.status`, `.body`) and show the person what happened.

SvelteKit 3 has no `$lib` alias, and this shell configures none: import with relative paths.

## Put a page in the nav

Add a sibling `nav.ts` to the route directory:

```ts
// web/src/routes/(app)/reports/nav.ts
import type { NavMeta } from '../../../lib/shell/types';
export const nav: NavMeta = { label: 'Reports', order: 20, group: 'data' };
```

- `label` is plain text, 1 to 64 characters: letters, digits, spaces and `. , & ( ) / _ : -`.
- `order` is an integer from 0 to 1000; lower is first. `group` is optional, a lower-case identifier.
- The link is the route's path with route groups removed, so the file above links to `/reports`.
- A route with a parameter (`[id]`) has no single link and cannot be a nav entry.
- **Put the nav entry in `nav.ts`, never in `+page.ts`:** SvelteKit rejects unknown exports from
  `+page.ts`, and the build fails.
- Delete `nav.ts` to take a page out of the nav. The shell collects every
  `src/routes/**/nav.ts` at build time through `import.meta.glob`; a malformed one fails the dev
  server with the file named, and is skipped with a warning in a production build.

## What the shell seeds (yours from the first compose)

| File | Role |
|---|---|
| `src/routes/+layout.svelte`, `+layout.ts` | The slot host: providers wrap the kit's `AppShell`; every navigation runs the route guard |
| `src/lib/shell/nav.generated.ts`, `providers.generated.ts` | Rendered from plugin slot fragments (`web.nav`, `web.providers`, `web.route-guard`) |
| `src/lib/shell/nav.ts`, `guard.ts`, `api.ts`, `types.ts`, `scope.ts` | Nav merge and ordering, guard runner, API client, contract types |
| `src/service-worker/` | Caches the built app so it opens offline; never caches `/api` |
| `src/lib/data-layer/`, `src/lib/data/`, `src/lib/sync/` (not `sync/collections.ts`) | Both data layers and the entry point the platform selected before Start: online (TanStack Query, the data-online skill) or offline-first (outbox and sync, the data-offline skill). `data-layer/index.ts` names the one in use |
| `vite.config.ts` (SvelteKit 3 config lives here), `tsconfig.json`, `scripts/local-serve.mjs` | Build and the local run |

Each is yours from the first compose. Prefer adding routes, nav entries and plugin fragments to
editing them, and edit one when the app needs it.

## Seeded files you own

`src/routes/+error.svelte` (error and 404), `src/routes/offline/+page.svelte`,
`src/routes/+page.svelte` and `nav.ts` (Home), `src/lib/app/app.css`,
`package.json` and its lockfile, and `src/lib/sync/collections.ts` (the synced collections, read
only by the offline data layer).

`src/routes/+page.svelte` is the home page stub: it carries the comment `<!-- albitor:home-stub -->`,
which the foundation gate looks for in the source and fails on until the real home page replaces
the file. Replace the whole page, marker included; never keep the comment to quiet a check.

`src/app.html` was written once from the `web.root-classes` slot and is yours: edit the `html` and
`body` classes there, and put head content (favicon, meta) in `<svelte:head>` in a layout.

## Plugins: providers and the route guard

A plugin that fills `web.providers` or `web.route-guard` names an identifier and ships a module in
`src/lib/shell/scope.d/` that exports it. A provider is a Svelte component that renders its
`children` snippet. A guard is `({ url }) => string | void`: return a same-origin path to redirect,
nothing to allow, and always allow the path you redirect to. A guard is a convenience, not access
control; the API must check the caller. The guard runs in `+layout.ts` `load`, before providers
mount, so it has no session-state ordering guarantee: do not assume an auth provider has restored
the session yet. Never persist tokens in `localStorage`.

A plugin `web.nav` entry only names a label, so the plugin must ship the route its label slugs to
(`Sign in` needs `/sign-in`); otherwise the link is dead. Two plugin entries slugging to one link
are an error, and so is a label with no letters or digits.

The slot files (`nav.generated.ts`, `providers.generated.ts`) were written once from the slot
fragments of the plugins composed with the shell. A plugin added later does not rewire them: add its
entry by hand.

## Run and verify

- `npm run dev` starts the dev server and forwards `/api` to `APP_API_ORIGIN` (default
  `http://127.0.0.1:8080`).
- `npm run local` builds and serves `build/` on `PORT` (default 3000) with `/api` forwarded and
  `/healthz` answering `ok`: the web side of the local run contract.
- `npm run check` (types), `npm test` (the shell's unit tests), `npm run build`.

## The component-kit contract

The shell imports `src/lib/kit/AppShell.svelte` from the component-kit plugin. It receives
`nav: NavGroup[]` (ungrouped entries first, then each group), `path: string` (the current
pathname, for the active link), `title: string` (the app's name, from the platform's `app_name`
token) and `children: Snippet`. It must render the landmarks: a nav, a `main` and a skip link.
