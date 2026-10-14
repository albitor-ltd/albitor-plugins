# client-offline

The `client-data@1` plugin for offline-first apps
([ADR-0090](https://github.com/albitor-ltd/albitor) section 2 and 13, albitor-ltd/albitor#3172). It
composes into a new app's `web/` beside `web-svelte`, in place of `client-online` (the socket takes
exactly one).

## What it provides

`@albitor/sync`, shipped as managed source under `web/src/lib/sync/` until a published package
exists (the same arrangement as the shell's API client):

- A local store per signed-in person (IndexedDB, one database each; an in-memory store when there is
  no IndexedDB), and an **outbox** of creates, updates and deletes replayed to the API when there is
  a network. Record ids are chosen on the device, so a replay is safe.
- One guarded run at a time: concurrent `flush()` and `sync()` callers share it, and a Web Lock
  keeps other tabs out. Work added during a run is picked up by the same run.
- Deletions travel as outbox entries up and as tombstones down, so other devices drop the record.
- Sign-out never discards unsynced work: `signOut()` uploads what it can and refuses, with a count,
  if anything is left. Only `signOut({ discard: true })` throws the queue away.
- Automatic sync after a write, on `online`, on tab focus and on a timer; a polite status line
  (`SyncStatus.svelte`); `liveCollection` and `liveStatus` reactive views.
- The shell's service worker (from `web-svelte`) precaches the app and falls back to the cached
  shell offline. This plugin's tests and compose check fail if the build stops shipping it.

## Defects it is built to keep out

Plantrain's regenerated sync layer carried these; each has a test that fails without the fix
(`engine.test.ts`):

| Defect | How it is excluded |
|---|---|
| An edit made while its create is in flight is lost | An in-flight entry is never amended; the edit is its own entry, and completion only removes the entry it sent (`rev` guard) |
| A different person after a lapsed session sees the last person's data, and the last person's queue uploads | Stores are per person; the engine only touches the attached person's store; the API transport refuses when the signed-in user is not the queue's owner |
| Deletions never reach other devices | Tombstones in the pull contract, applied unless the record has unsent local changes |
| Sign-out silently discards unsynced work | `signOut()` refuses while anything is queued or failed |
| Concurrent flushes bypass the engine's guard | Single-flight per person, plus a cross-tab lock |

## What it needs

- `web-framework@1` (`web-svelte`): the slot host, the API client and the service worker.
- From the **auth** plugin: a call to `setSyncUser(userId)` when a session starts, changes or ends
  (`lib/sync/session.ts`). The id is the stable user id, never an email.
- From the **API**: the routes in `lib/sync/transport.ts` (create with a client id, partial update,
  delete with a tombstone, and a `?since=` pull returning `{items, deleted, cursor}`). A repeated
  create answers 409.
- The app lists its collections in `lib/sync/collections.ts` (an app-owned example file).

## Known gap

`fake-indexeddb` (test only) is recorded as a dependency but the composer does not yet write it into
the app's `web/package.json` (albitor-ltd/albitor#3235). `scripts/compose-check.sh` installs it
before building the composed tree.

## Layout

```text
.claude-plugin/plugin.json   schema-2 manifest
templates/web/**             the files composed into the app, at their repository paths
skills/client-offline/       the skill
```

Every file is `managed` except `collections.ts`. Its tests (`*.test.ts`) ship with it and run in the
app's `npm test`.
