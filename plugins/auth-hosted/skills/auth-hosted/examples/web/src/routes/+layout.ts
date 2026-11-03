// A single-page app: no server rendering, and every navigation
// first runs the web.route-guard guard, which may redirect. It replaces the shell's own
// +layout.ts: the one change is that the app's navigation is shown only to someone signed in.
import { redirect } from '@sveltejs/kit';
import { canShowNav } from '../lib/auth/guard';
import { runGuards } from '../lib/shell/guard';
import { routeGuards } from '../lib/shell/providers.generated';
import type { LayoutLoad } from './$types';

export const ssr = false;
export const prerender = false;
export const trailingSlash = 'never';

export const load: LayoutLoad = async ({ url }) => {
	const to = await runGuards(routeGuards, url);
	if (to !== null) redirect(307, to);
	// The public screens are reached signed out: they get the shell without the app's links, and
	// still render when the API is down (canShowNav fails closed). getSession() is cached.
	return { showNav: await canShowNav(url) };
};
