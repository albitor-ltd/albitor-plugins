// Managed by Albitor (web-svelte). A single-page app: no server rendering, and every navigation
// first runs the web.route-guard guard, which may redirect.
import { redirect } from '@sveltejs/kit';
import { runGuards } from '../lib/shell/guard';
import { routeGuards } from '../lib/shell/providers.generated';
import type { LayoutLoad } from './$types';

export const ssr = false;
export const prerender = false;
export const trailingSlash = 'never';

export const load: LayoutLoad = async ({ url }) => {
	const to = await runGuards(routeGuards, url);
	if (to !== null) redirect(307, to);
};
