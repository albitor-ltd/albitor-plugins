// The data layer this app uses: online-only TanStack Query over the app's own API (lib/data).
// The shell's DataProvider slot (lib/shell/scope.d/data-layer.ts) reads this file, so a change of
// layer is a change here: export the other layer's DataProvider and DATA_LAYER from this file.
export { DataProvider, DATA_LAYER } from './online';
