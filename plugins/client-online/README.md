# client-online

The `client-data@1` plugin for online-only apps
([ADR-0090](https://github.com/albitor-ltd/albitor) section 2 and 13, albitor-ltd/albitor#3171). It
composes into a new app's `web/` beside `web-svelte`.

## What it provides

- TanStack Query for Svelte (`@tanstack/svelte-query` 6.3.0, pinned exactly) behind the `web.providers` slot: a
  `QueryProvider` the shell nests around the app, with one query client per page load.
- A retry policy shared by every query: server errors, timeouts, throttling and an unreachable API
  retry twice with backoff; client errors never retry; an offline browser pauses the query rather
  than failing it. Mutations are never retried.
- `apiQuery(path, query)`: query options that go through the shell's API client, so path checks, the
  bearer token and request cancellation come for free.
- `DataState.svelte` and `viewState`: loading, offline, error (with retry), empty and ready, each
  replaceable by a snippet. `errorMessage` never repeats a response body.
- The `client-online` skill: how to read, write and invalidate.

## What it needs

- `web-framework@1` (`web-svelte`): the slot host (`web.providers`, `scope.d/`) and
  `src/lib/shell/api.ts`. It fills no slot other than `web.providers`.

## Known gap

The composer records `@tanstack/svelte-query` as a dependency but does not write it into the app's
`web/package.json`, which `web-svelte` seeds and the app owns. Until it does, `scripts/compose-check.sh`
installs the declared dependency before it builds the composed tree, and an app composed by the
platform needs that line added (tracked with the composer, not here).

## Layout

```text
.claude-plugin/plugin.json   schema-2 manifest
templates/web/**             the files composed into the app, at their repository paths
skills/client-online/        the skill
```

Every file is `managed`: a plugin upgrade re-renders it. Its tests (`*.test.ts`) ship with it and run
in the app's `npm test`.
