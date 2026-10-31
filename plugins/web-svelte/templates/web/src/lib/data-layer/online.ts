// Starting point (web-svelte): the online-only data layer's entry point. TanStack Query over
// the app's own API (lib/data); nothing is stored on the device.
export { default as DataProvider } from '../data/QueryProvider.svelte';

/** Which data layer this entry point is. */
export const DATA_LAYER = 'online' as const;
