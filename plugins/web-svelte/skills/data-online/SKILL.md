---
name: data-online
description: How to fetch and cache data from the app's own API on a SvelteKit app using the web-svelte online data layer (TanStack Query), and how to show loading, offline, error and empty states. Use when a build adds or changes a page that reads or writes API data.
---

# Online data (web-svelte, online data layer)

Data comes from this app's own `/api` through TanStack Query for Svelte. The shell already wraps the
app in the query client (the `DataProvider` in `web.providers` is `QueryProvider`), so a page only
asks for data. Nothing is stored on the device: this is the online-only data layer, and
`lib/data-layer/index.ts` exports it. An app that must work offline copies the offline data layer in
instead (the data-offline skill).

## Read data

```svelte
<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import DataState from '../../lib/data/DataState.svelte';
	import { apiQuery } from '../../lib/data/query';

	type Item = { id: string; name: string };
	const items = createQuery(() => apiQuery<{ items: Item[] }>('/items', { limit: 20 }));
</script>

<DataState query={items}>
	{#snippet children(data)}
		<ul>{#each (data as { items: Item[] }).items as item (item.id)}<li>{item.name}</li>{/each}</ul>
	{/snippet}
</DataState>
```

- `createQuery` takes a function, so options can depend on reactive values; the query re-runs when
  they change.
- `apiQuery(path, query)` goes through `lib/shell/api.ts`: the path must be app-relative, the bearer
  token is attached there, and the request is cancelled when the query is. Never `fetch` an API
  path directly.
- Keys are `['api', path, query]`. Use the same call everywhere you want the same cache entry.

## The five states

`DataState` renders them for you: loading, offline (the browser has no network and the first
fetch is waiting), error (with a "Try again" button), empty, and ready. Replace any with a snippet
(`loading`, `offline`, `error`, `empty`) when the default does not fit. An empty list is not a
failure, so give it its own words. If a refetch fails while data is on screen, the data stays.
`isEmpty` overrides what counts as empty; the default is no items (`[]` or `{ items: [] }`).

`errorMessage(error)` gives plain wording and never repeats the response body or an error's own
message. Do not render those yourself: they can carry server internals.

## Write data

```ts
import { createMutation, useQueryClient } from '@tanstack/svelte-query';
import { API_KEY } from '../../lib/data/query';
import { post } from '../../lib/shell/api';

const client = useQueryClient();
const add = createMutation(() => ({
	mutationFn: (name: string) => post('/items', { name }),
	onSuccess: () => client.invalidateQueries({ queryKey: [API_KEY, '/items'] })
}));
```

Mutations are never retried automatically. After a write, invalidate the lists it changes.

## Rules that stay

- One screen, one API call. A screen gets what it renders from a single call: a list endpoint
  returns the fields the list renders, a detail page makes one call. Never run a query per row or
  `Promise.all` over a list's ids. When a screen needs several entities, add a screen-shaped
  endpoint that aggregates server-side. A hosted app's API runs at most 10 executions at once and
  its stage takes 50 requests a second (100 in a burst), so a 14-row fan-out is throttled and the
  page errors.

  ```ts
  // Wrong: one request per row.
  const rows = await Promise.all(ids.map((id) => get(`/opportunities/${id}`)));
  // Right: the list endpoint already carries what the list shows.
  const { items } = await get<{ items: OpportunitySummary[] }>('/opportunities');
  ```
- Retries: `api()` retries a GET or HEAD that gets 429 or 503, or an unreachable API, up to three
  attempts in all, with jittered backoff never sooner than `Retry-After`; a `Retry-After` over 5
  seconds is not waited for. It never retries a write. It also holds at most `MAX_IN_FLIGHT` (6)
  requests in flight per tab and queues the rest, which is a safety net, not a licence to fan out.
  A failure `api()` already retried is final; the query layer does not retry it again. Other
  server errors, timeouts and an unreachable API retry twice with backoff; client errors (400 to
  499) never do. Change the policy in `lib/data/client.ts` and `lib/shell/api.ts`.
- On sign-out call `useQueryClient().clear()`, registered with `onBeforeSignOut` (`lib/shell/signout.ts`, awaited by the Sign out button before it posts), or the next person on the browser sees cached data.
- The cache lives in memory and is gone on reload. Do not copy API data into `localStorage`.
- A route guard hides pages, it does not protect data: the API must check the caller on every call.
