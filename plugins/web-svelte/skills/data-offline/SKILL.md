---
name: data-offline
description: How to read and write data that must work offline on a SvelteKit app using the web-svelte offline data layer (lib/sync): how to make an app offline by copying the worked example in, collections, the outbox, sync status, sign-out, and what the API must provide. Use when an app must work offline, or a build adds or changes a page that reads or writes synced data.
---

# Offline-first data (web-svelte, offline data layer)

Records live on the device first. A write lands in the local store and in an outbox, the page
updates at once, and the outbox is sent to the API whenever there is a network. Reads never wait
for the network. An online-only app uses the data-online skill, and the seeded data layer is that
one. Do not use TanStack Query (`lib/data`) for synced collections.

## Make the app offline

The offline layer is a worked example, not seeded code. It lives in this skill at
`examples/web/src/lib/` (the `data-offline` example), laid out as the app's own paths:

- `sync/**` (engine, stores, transport, auto-sync, status, `SyncProvider.svelte`,
  `SyncStatus.svelte`, `collections.ts`, and its tests)
- `data-layer/offline.ts` (the entry point, exporting `SyncProvider` as `DataProvider`)

1. Copy `examples/web/src/lib/sync/` and `examples/web/src/lib/data-layer/offline.ts` to the same
   paths under the app's `web/src/lib/`.
2. Swap the entry point: in `web/src/lib/data-layer/index.ts` change `'./online'` to `'./offline'`.
   Nothing else imports a layer, and the shell's `scope.d/data-layer.ts` already follows `index.ts`.
3. Declare the collections, wire `setSyncUser` (below) and put `<SyncStatus />` in the layout.
4. Keep the copied tests green: `fake-indexeddb` is already in `package.json` for them, and
   `sync/shell.test.ts` reads the seeded service worker, which must stay.

## Declare the collection

Add each kind of synced record to `src/lib/sync/collections.ts` (the app's, once copied in):

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
- **No auth plugin makes that call.** Wire it in the app's own code where the session is read (for
  `auth-cognito`, beside `getSession()`; for `auth-none`, once at start-up with its fixed owner id).
  Without it the engine never attaches, and nothing syncs.
- Before ending a session, `await sync.signOut()`. If it returns `{ ok: false, pending }`, tell the
  person that many changes have not been saved and let them wait, or choose `signOut({ discard: true })`
  explicitly. Never discard silently.

## Rules that stay

- Once copied in, `lib/sync/*` is the app's code: `collections.ts` is where the app lists its
  collections; change the rest only when the app needs it, and keep `engine.test.ts` and
  `shell.test.ts` passing.
- Do not unregister or bypass the service worker (`src/service-worker`): it is what lets the shell
  open offline, and it deliberately never caches `/api/`.
- A route guard hides pages, it does not protect data: the API must check the caller on every call.
- Conflicts are last-write-wins per request. If two devices edit the same field offline, the later
  upload wins; say so in the design if that matters.
