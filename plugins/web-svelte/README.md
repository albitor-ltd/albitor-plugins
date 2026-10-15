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
- Both client data layers, and the entry point the platform selects (below), with the
  `data-online` and `data-offline` skills.
- The `svelte-shell` skill: how to add a route, a page and a nav entry.

## The data layer: online or offline-first

The client data layer was the `client-data@1` socket with two plugins, `client-online` and
`client-offline`. It is now part of this plugin (albitor-ltd/albitor ADR-0090, amendment of
2026-10-15), so an app that must work offline needs no extra plugin registered. Both layers ship in
every app; the platform decides which one is live **before Start**, from the plan's `offline`
capability need, and passes it as the `data_layer` platform token (`online` or `offline`, nothing
else). One managed file reads it:

```ts
// web/src/lib/data-layer/index.ts
export { DataProvider, DATA_LAYER } from './{{albitor.data_layer}}';
```

`DataProvider` is this plugin's own `web.providers` fragment (`scope.d/data-layer.ts`), so the
unselected layer is never imported and never reaches the bundle. Changing layer is a recompose,
not an edit.

- **Online** (`src/lib/data/`, was `client-online`): TanStack Query for Svelte
  (`@tanstack/svelte-query` 6.3.0) with one query client per page load; a shared retry policy
  (server errors, timeouts, throttling and an unreachable API retry twice with backoff; client
  errors never; an offline browser pauses rather than fails; mutations never retry); `apiQuery`
  through the shell's API client; `DataState.svelte` and `viewState` for loading, offline, error,
  empty and ready, with `errorMessage` never repeating a response body.
- **Offline-first** (`src/lib/sync/`, was `client-offline`; `@albitor/sync` as managed source until
  a published package exists): a local store per signed-in person (IndexedDB, in memory without
  it), an outbox of creates, updates and deletes replayed when there is a network, device-chosen
  ids, one guarded run at a time (and a Web Lock across tabs), tombstones for deletions, failed
  changes after 5 backed-off tries visible to the person, sign-out that never discards unsynced
  work, automatic sync and a status line. `collections.ts` is the app's (example mode).

The offline layer needs, from the **auth** plugin, a call to `setSyncUser(userId)`
(`lib/sync/session.ts`) when a session starts, changes or ends, before the bearer token source
changes; and from the **API**, the routes in `lib/sync/transport.ts` (create with a client id,
partial update, delete with a tombstone, a `?since=` pull returning `{items, deleted, cursor}`).

Defects of plantrain's regenerated sync layer that `engine.test.ts` keeps out: an edit made while
its create is in flight being lost; a different person after a lapsed session seeing or uploading
the last person's data; deletions never reaching other devices; sign-out silently discarding
unsynced work; concurrent flushes bypassing the guard.

**Known gap: no auth plugin calls `setSyncUser` yet.** Neither `auth-cognito` nor `auth-none`
wires it, so in a composed offline app the sync engine never attaches to a person, and so never
syncs, until the build wires the call (the data-offline skill says where). `lib/sync/session.ts`
ships in every app, so the auth plugins can import it unconditionally; that is a follow-up there.

`DataProvider` is the outermost provider: slot fragments merge in socket order and this plugin's
socket comes first. No other plugin fills `web.providers` today; a provider whose context the data
layer must read could not wrap it.

Known gaps of the offline layer, on purpose or yet: no caps on record, payload, outbox or
pull-page size, and an unpaginated pull; no `BroadcastChannel`; create and update use two storage
transactions; discarding during a push can write the entry back as failed; with no usable IndexedDB
data is in memory and lost on reload; a permanently failed change holds its collection's pull
cursor back; a server that hangs for ever pauses the queue without counting attempts.

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

## The lockfile is the app's

The plugin seeds `web/package-lock.json` only as a starting point. The composer omits it whenever
the npm merge changes `web/package.json`, so the app owns the lockfile from then on. The shipped
build command is therefore `cd web && npm install && npm run build`, not `npm ci`: there is no
plugin-supplied lockfile for `npm ci` to be reproducible against. Commit the lockfile `npm install`
writes.

## What it needs from other plugins

- `component-kit@1`: a `web/src/lib/kit/AppShell.svelte` taking `nav: NavGroup[]`,
  `path: string` and `children: Snippet` (see the skill).
- `deploy-target@1`: only because this plugin contributes the npm entry for `repo.dependabot`.
- From the platform: the `data_layer` token. A composer that does not own it refuses the
  composition (`unknown_token`), so this version needs an albitor composer with the token.

## Layout

```text
.claude-plugin/plugin.json   schema-2 manifest
templates/web/**             the files composed into the app, at their repository paths; the
                             two generated slot files (src/lib/shell/nav.generated.ts and
                             providers.generated.ts) are slot-template entries in plugin.json
skills/svelte-shell/         the shell skill
skills/data-online/          the online data layer skill
skills/data-offline/         the offline data layer skill
```

Slot templates are declared in the manifest (`mode: "slot-template"`, `syntax: "ts"`; ADR-0090
amendment of 2026-10-14, item 1); the old `slot-templates.json` sidecar is gone, because the
platform never read it (dev build 63716a5c refused to compose with `no_slot_template`). A route guard is a convenience, not access control: the API must check
the caller. The guard runs in `+layout.ts` `load` before providers mount, so there is no
session-state ordering guarantee, and tokens must not be persisted in `localStorage`. A plugin
`web.nav` entry must ship the route its label slugs to.

Compose, build and test it with `scripts/compose-check.sh` (see the repository README).
