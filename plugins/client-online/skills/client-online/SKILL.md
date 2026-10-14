---
name: client-online
description: How to fetch and cache data from the app's own API on a SvelteKit app composed with the client-online plugin (TanStack Query), and how to show loading, offline, error and empty states. Use when a build adds or changes a page that reads or writes API data.
---

# Online data with client-online

Data comes from this app's own `/api` through TanStack Query for Svelte. The shell already wraps the
app in the query client (the `QueryProvider` in `web.providers`), so a page only asks for data.
Nothing is stored on the device: this is the online-only choice for `client-data`. An app that must
work offline needs `client-offline` instead.

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

- Retries: server errors, timeouts, throttling and an unreachable API retry twice with backoff;
  client errors (400 to 499) never do. Change the policy in `lib/data/client.ts` only through the
  plugin, since the file is managed.
- On sign-out call `useQueryClient().clear()`, or the next person on the browser sees cached data.
- The cache lives in memory and is gone on reload. Do not copy API data into `localStorage`.
- A route guard hides pages, it does not protect data: the API must check the caller on every call.
