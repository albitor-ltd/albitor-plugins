// Starting point (auth-hosted): the calls the sign-in, sign-out and account screens make.
//
// There is no password in this app. A person signs in on the app's Cognito hosted page: the browser
// is sent to the API's GET /api/auth/login, which redirects to Cognito, and Cognito sends it back to
// the API's callback, which sets the session as HttpOnly cookies. So no token is ever in reach of
// page script, and the browser makes no call to Cognito.
//
// The two POST calls answer a result and never throw on a refusal, so a screen shows what happened
// instead of an error page.
import { API_BASE, ApiError, NetworkError, post } from '../shell/api';
import { clearSession } from './session';

/** What the screens can act on. `code` is stable; `message` is safe to show. */
export type AuthResult<T = object> =
	| ({ ok: true } & T)
	| { ok: false; code: string; message: string };

const UNAVAILABLE = { ok: false, code: 'unavailable', message: 'Sign-in is unavailable: try again shortly.' } as const;
const OFFLINE = { ok: false, code: 'offline', message: 'You are offline: check your connection and try again.' } as const;

async function call<T extends object>(action: string): Promise<AuthResult<T>> {
	try {
		const out = await post<T | undefined>(`/auth/${action}`, {});
		return { ok: true, ...(out ?? ({} as T)) };
	} catch (err) {
		if (err instanceof NetworkError) return err.offline ? OFFLINE : UNAVAILABLE;
		if (err instanceof ApiError) {
			const b = (err.body ?? {}) as { code?: unknown; error?: unknown };
			// 501 is a local run saying it has no hosted sign-in: a stable code, not a fault.
			if ((err.status < 500 || err.status === 501) && typeof b.code === 'string' && typeof b.error === 'string') {
				return { ok: false, code: b.code, message: b.error };
			}
			return UNAVAILABLE;
		}
		throw err;
	}
}

/**
 * Where "sign in" goes: the API starts the sign-in and sends the browser to Cognito. `next` is where
 * the person lands afterwards; the API accepts a path on this site only, and anything else becomes `/`.
 */
export function loginUrl(next?: string | null): string {
	return next ? `${API_BASE}/auth/login?next=${encodeURIComponent(next)}` : `${API_BASE}/auth/login`;
}

/** The messages the sign-in page shows when the API sent a person back with `?error=<code>`. */
export const SIGN_IN_ERRORS: Readonly<Record<string, string>> = {
	denied: 'Sign-in was cancelled or could not be completed.',
	expired: 'Your sign-in took too long or was interrupted.',
	failed: 'We could not finish signing you in.',
	unverified: 'Your email address has not been verified.',
	toolarge: 'Your sign-in details are too large for this app to keep.',
	unavailable: 'Sign-in is unavailable: try again shortly.',
	local: 'Sign-in is not available in a local run: use a local test identity.'
};

/** The text for an `?error=` code, or a general one for a code it does not know. */
export function signInErrorMessage(code: string): string {
	return SIGN_IN_ERRORS[code] ?? SIGN_IN_ERRORS.failed!;
}

/** True for a URL the API may send a person to: Cognito's hosted pages, over https. */
export function isHostedUrl(url: unknown): url is string {
	if (typeof url !== 'string') return false;
	try {
		return new URL(url).protocol === 'https:';
	} catch {
		return false;
	}
}

/**
 * Signs out: the API revokes the refresh token, clears the cookies and answers `logout_url`, the
 * Cognito page to navigate to so Cognito's own cookie is cleared too (without it, the next sign-in
 * would not ask for a password). Absent in a local run, which has no Cognito.
 */
export async function signOut(): Promise<AuthResult<{ logout_url?: string }>> {
	const result = await call<{ logout_url?: string }>('sign-out');
	clearSession();
	return result;
}

/**
 * Deletes the signed-in person's account. On success the API has cleared the cookies and answers
 * `logout_url`, as `signOut` does. `scope_missing` means the app's sign-in is not set up to allow it.
 */
export async function deleteAccount(): Promise<AuthResult<{ logout_url?: string }>> {
	const result = await call<{ logout_url?: string }>('delete');
	if (result.ok) clearSession();
	return result;
}
