// Starting point (auth-hosted): the route guard the shell runs before every navigation.
//
// It is a function over the session module, never over a provider: `load` runs before any provider
// mounts. A guard shapes what the browser shows; the API enforces access itself.
import { isSafeRedirect } from '../shell/guard';
import type { Guard } from '../shell/types';
import { getSession, takeRefusal } from './session';

/** Paths a signed-out person must reach, or the guard would loop. `/offline` cannot ask the API. */
export const PUBLIC_PATHS: readonly string[] = ['/sign-in', '/sign-out', '/not-allowed', '/offline'];

export function isPublicPath(pathname: string): boolean {
	return PUBLIC_PATHS.includes(pathname);
}

/** Where a signed-out person goes, remembering where they were headed. */
export function signInPath(url: URL): string {
	return `/sign-in?next=${encodeURIComponent(url.pathname + url.search)}`;
}

/** The place to send a person after they sign in: `next` when it is a same-origin path, else `/`. */
export function nextPath(next: string | null | undefined): string {
	return next && isSafeRedirect(next) && !isPublicPath(next.split(/[?#]/)[0]!) ? next : '/';
}

/** Sends a signed-out person to the sign-in screen, and lets everyone else through. */
export const requireSession: Guard = async ({ url }) => {
	if (isPublicPath(url.pathname)) return;
	if (await getSession()) return;
	// The API refused the address on a refresh: say so, rather than loop through sign-in.
	if (takeRefusal()) return '/not-allowed';
	return signInPath(url);
};
