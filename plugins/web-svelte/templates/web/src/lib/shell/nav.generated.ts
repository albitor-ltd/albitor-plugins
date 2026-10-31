// Starting point (web-svelte): the navigation entries the plugins composed with the shell declared,
// written once. It is yours from here on; routes add their own entries through a nav.ts beside them.
// Add a page to the nav with a sibling nav.ts, not here (see the svelte-shell skill).
import { buildNav } from './nav';
import type { NavMeta } from './types';

/**
 * Entries contributed by plugins through the web.nav slot. With no contribution the composer
 * renders the placeholder as nothing, not as [], so concat takes zero arguments then.
 */
const pluginNav = ([] as NavMeta[]).concat({{albitor.slot.web.nav}});

/** Every app-owned nav.ts under src/routes, collected at build time. */
const routeNavModules = import.meta.glob('/src/routes/**/nav.ts', { eager: true });

export const navGroups = buildNav(pluginNav, routeNavModules);
