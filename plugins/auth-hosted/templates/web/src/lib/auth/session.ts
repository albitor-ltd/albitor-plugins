// Managed by Albitor (auth-hosted): the session, read without any provider.
//
// SvelteKit runs the route guard in `+layout.ts` `load`, before any provider mounts, so the guard
// cannot read a context a provider sets. This module is what the guard and the screens both read,
// and it depends on nothing in the page: the session lives in HttpOnly cookies the page cannot see,
// so the only way to know who is signed in is to ask the API, GET /api/me.
//
// It must not touch `window` unguarded: `load` also runs on the server.
import { ApiError, get } from '../shell/api';

/** The person a request acts for, as GET /api/me answers it. */
export interface Session {
	sub: string;
	email?: string;
	groups?: string[];
	/** True in a local run with no sign-in at all (the API's anonymous owner). */
	local?: boolean;
}

/** How long an answer is reused, so a burst of navigations asks the API once. */
export const SESSION_TTL_MS = 10_000;

interface Cached {
	at: number;
	session: Session | null;
}

let cached: Cached | null = null;
let refused = false;

/** Whether the API refused the person's address (the sign-up domain rule), once. The guard sends them to the refusal page. */
export function takeRefusal(): boolean {
	const was = refused;
	refused = false;
	return was;
}
let inFlight: Promise<Session | null> | null = null;

/** Forget the cached answer. Sign-in and sign-out call it; so should anything that changes who the
 * cookies belong to. */
export function clearSession(): void {
	cached = null;
	inFlight = null;
}

/**
 * Who is signed in: the session, or null when nobody is. A 401 is "nobody"; any other failure (the
 * API is down, the network is) is thrown, never read as "signed out", so a blip does not bounce a
 * signed-in person to the sign-in screen. Server-side there is no session to read: null.
 */
export function getSession(): Promise<Session | null> {
	if (typeof window === 'undefined') return Promise.resolve(null);
	if (cached && Date.now() - cached.at < SESSION_TTL_MS) return Promise.resolve(cached.session);
	if (inFlight) return inFlight;

	const request = (async () => {
		try {
			const session = await get<Session>('/me');
			return session && typeof session.sub === 'string' && session.sub !== '' ? session : null;
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				const b = (err.body ?? {}) as { code?: unknown };
				if (b.code === 'email_not_allowed') refused = true;
				return null;
			}
			throw err;
		}
	})();
	const pending = request
		.then((session) => {
			// A clearSession() during the call means this answer may be for the previous person.
			if (inFlight === pending) cached = { at: Date.now(), session };
			return session;
		})
		.finally(() => {
			if (inFlight === pending) inFlight = null;
		});
	inFlight = pending;
	return pending;
}
