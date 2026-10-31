// Starting point (web-svelte): the providers and route guards the plugins composed with the shell
// declared, written once. It is yours from here on: add or remove entries as the app needs.
// Identifiers resolve against the scope in ./scope (see the
// svelte-shell skill).
import { checkGuards, checkProviders } from './guard';
import { scope } from './scope';

const { {{albitor.slot.web.providers}} } = scope;
const { {{albitor.slot.web.route-guard}} } = scope;

/** Provider components, outermost first. */
export const providers = checkProviders([{{albitor.slot.web.providers}}]);

/** Route guards. The slot is single-valued, so this holds zero or one. */
export const routeGuards = checkGuards([{{albitor.slot.web.route-guard}}]);
