// Starting point (web-svelte): the offline-first data layer's entry point. A local store per
// signed-in person with an outbox and sync (lib/sync).
export { default as DataProvider } from '../sync/SyncProvider.svelte';

/** Which data layer this entry point is. */
export const DATA_LAYER = 'offline' as const;
