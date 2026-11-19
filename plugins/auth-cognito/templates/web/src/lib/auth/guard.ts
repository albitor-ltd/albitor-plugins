// The route guard the shell runs before every navigation.
//
// It is a function over the session module, never over a provider: `load` runs before any provider
// mounts. A guard shapes what the browser shows; the API enforces access itself.
import { isSafeRedirect } from '../shell/guard';
import type { Guard } from '../shell/types';
import { getSession } from './session';

/**
 * The paths opened to a signed-out person by declaration: the sign-in and account pages first, then
 * (in posture public) the web entries of `services.auth.public_routes`. It stays ONE literal array
 * of strings: the floor's manifest check reads it, requires every manifest web entry in it and
 * allows no other entry unless it is an auth or account page, so never build it from spreads,
 * imports or calls, and add nothing the manifest does not list. An entry is an exact path, or a
 * prefix that ends in `/*` (`/listings/*` matches `/listings/42` but not `/listings`; list that
 * separately). The foundation node adds the manifest's web entries; nothing else adds here.
 */
export const PUBLIC_PATHS: readonly string[] = ['/sign-in', '/sign-up', '/confirm'];

/**
 * The auth pages: reachable signed out, never given the app's nav, and never a `next` target. A
 * superset of the auth entries in PUBLIC_PATHS: the screens the floor does not name (the refusal,
 * the reset and the offline page) are open here only.
 */
export const AUTH_PAGES: readonly string[] = [
	'/sign-in',
	'/sign-up',
	'/confirm',
	'/reset',
	'/offline'
];

function matches(entry: string, pathname: string): boolean {
	if (!entry.endsWith('/*')) return pathname === entry;
	const prefix = entry.slice(0, -1);
	return pathname.startsWith(prefix) && pathname.length > prefix.length;
}

export function isPublicPath(pathname: string): boolean {
	return isAuthPage(pathname) || PUBLIC_PATHS.some((entry) => matches(entry, pathname));
}

export function isAuthPage(pathname: string): boolean {
	return AUTH_PAGES.includes(pathname);
}

/** Where a signed-out person goes, remembering where they were headed. */
export function signInPath(url: URL): string {
	return `/sign-in?next=${encodeURIComponent(url.pathname + url.search)}`;
}

/** The place to send a person after they sign in: `next` when it is a same-origin path, else `/`. */
export function nextPath(next: string | null | undefined): string {
	return next && isSafeRedirect(next) && !isAuthPage(next.split(/[?#]/)[0]!) ? next : '/';
}

/** Sends a signed-out person to the sign-in screen, and lets everyone else through. */
export const requireSession: Guard = async ({ url }) => {
	if (isPublicPath(url.pathname)) return;
	if (await getSession()) return;
	return signInPath(url);
};
