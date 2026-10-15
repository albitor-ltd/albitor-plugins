---
name: data-offline
description: How to read and write data that must work offline on a SvelteKit app composed with the web-svelte offline data layer (@albitor/sync): collections, the outbox, sync status, sign-out, and what the API must provide. Use when a build adds or changes a page that reads or writes synced data.
---

# Offline-first data (web-svelte, offline data layer)

Records live on the device first. A write lands in the local store and in an outbox, the page
updates at once, and the outbox is sent to the API whenever there is a network. Reads never wait
for the network. This is the data layer an app is composed with when its plan has the `offline`
need (`lib/data-layer/index.ts` says which, and its `DataProvider` is `SyncProvider`); an
online-only app uses the data-online skill. Do not use TanStack Query (`lib/data`) for synced
collections: it ships in every app but is not imported unless the online layer is selected.

## Declare the collection

Add each kind of synced record to `src/lib/sync/collections.ts` (app-owned):

```ts
export const collections = ['todos'] as const;
```

The API must provide, for each collection (see the header of `src/lib/sync/transport.ts`): create
(`POST /todos` with the device's `id`; a repeat answers 409), partial update (`PUT /todos/:id`),
delete (`DELETE /todos/:id`, keeping a tombstone) and `GET /todos?since=C` returning
`{ items, deleted, cursor }`. Without the tombstones, deletions never reach other devices.

## Read and write

```svelte
<script lang="ts">
	import { getSync } from '../../lib/sync/context';
	import { liveCollection } from '../../lib/sync/live.svelte';

	type Todo = { id: string; title: string; done?: boolean };
	const sync = getSync();
	const todos = liveCollection<Todo>(sync, 'todos');
</script>

{#each todos.items as todo (todo.id)}
	<label>
		<input type="checkbox" checked={todo.done} onchange={() => sync.update('todos', todo.id, { done: !todo.done })} />
		{todo.title}
	</label>
	<button onclick={() => sync.remove('todos', todo.id)}>Delete</button>
{/each}
<button onclick={() => sync.create('todos', { title: 'New' })}>Add</button>
```

- `create` returns the record with its new id; `update` takes only the changed fields; `remove`
  deletes locally and queues the delete. All three resolve once the change is on the device.
- Record ids use letters, digits, `-` and `_` (up to 64). Let `create` generate them.
- Never `fetch` an API path for these collections, and never copy them into `localStorage`.
- Syncing is automatic (after writes, on reconnect, on focus, every 30 seconds). Call
  `sync.sync()` only to force it; concurrent calls share one run.

## Show the state

Put `<SyncStatus />` (`lib/sync/SyncStatus.svelte`) in the layout. It says in plain words when
changes are waiting, when the person is offline, when a session needs signing in again, and when a
change was rejected. For rejected changes, `sync.failures()` lists them (no server text), and
`sync.retryFailed()` or `sync.discardFailed(id)` resolves each; offer both.

## Sign-in and sign-out

- The auth integration calls `setSyncUser(userId)` (`lib/sync/session.ts`) with the stable user id
  when a session starts or the person changes, and `setSyncUser(null)` when it ends. Call it before the bearer token source switches to the new person (and before it is cleared). Each person has
  their own local database; one never sees or uploads another's.
- Before ending a session, `await sync.signOut()`. If it returns `{ ok: false, pending }`, tell the
  person that many changes have not been saved and let them wait, or choose `signOut({ discard: true })`
  explicitly. Never discard silently.

## Rules that stay

- Do not edit `lib/sync/*` except `collections.ts`; it is managed and upgraded with the plugin.
- Do not unregister or bypass the service worker (`src/service-worker`): it is what lets the shell
  open offline, and it deliberately never caches `/api/`.
- A route guard hides pages, it does not protect data: the API must check the caller on every call.
- Conflicts are last-write-wins per request. If two devices edit the same field offline, the later
  upload wins; say so in the design if that matters.
