// Starting point (web-svelte): the data layer this app was composed with. The platform decides
// it before Start from the plan's `offline` capability and renders it here (the data_layer token):
// `online` is TanStack Query (lib/data), `offline` is the outbox and sync (lib/sync). Both ship;
// only the one named here is imported, so the other never reaches the bundle. Changing layer is a
// recompose, not an edit (see the data-online and data-offline skills).
export { DataProvider, DATA_LAYER } from './{{albitor.data_layer}}';
