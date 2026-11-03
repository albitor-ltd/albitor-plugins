// The route guard the shell runs before every navigation.
//
// It is a function over the session module, never over a provider: `load` runs before any provider
// mounts. A guard shapes what the browser shows; the API enforces access itself.
import { isSafeRedirect } from '../shell/guard';
import type { Guard } from '../shell/types';
import { getSignUpMode } from './client';
import { getSession } from './session';

/** Paths a signed-out person must reach, or the guard would loop. `/offline` cannot ask the API. */
export const PUBLIC_PATHS: readonly string[] = [
	'/sign-in',
	'/sign-up',
	'/confirm',
	'/reset',
	'/offline'
];

/** Screens only an app that lets people register themselves has; an invite-only app sends them to sign-in. */
export const SELF_SERVICE_PATHS: readonly string[] = ['/sign-up', '/confirm', '/reset'];

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

/**
 * Whether the shell shows the app's navigation: only to someone signed in. It fails closed and
 * never throws, so an API that is down or an offline device still renders the public screens
 * (`/sign-in`, `/offline`...), just without the nav. Public paths are not asked about at all.
 */
export async function canShowNav(url: URL): Promise<boolean> {
	if (isPublicPath(url.pathname)) return false;
	try {
		return (await getSession()) !== null;
	} catch {
		return false;
	}
}

/** Sends a signed-out person to the sign-in screen, and lets everyone else through. */
export const requireSession: Guard = async ({ url }) => {
	if (SELF_SERVICE_PATHS.includes(url.pathname) && (await getSignUpMode()) === 'invite_only') {
		return '/sign-in';
	}
	if (isPublicPath(url.pathname)) return;
	if (await getSession()) return;
	return signInPath(url);
};
