// Starting point (auth-cognito): the calls the sign-in, sign-up, confirm and reset screens make.
//
// The browser never talks to Cognito. It posts to the API's /api/auth/*, which calls Cognito and
// answers with a stable `code`; the session comes back as HttpOnly cookies, so no token is ever in
// reach of page script. Every call answers a result and never throws on a refusal, so a screen
// shows what happened instead of an error page.
import { ApiError, NetworkError, get, post } from '../shell/api';
import { clearSession } from './session';

/** What the screens can act on. `code` is stable; `message` is safe to show. */
export type AuthResult<T = object> =
	| ({ ok: true } & T)
	| { ok: false; code: string; message: string; session?: string };

const UNAVAILABLE = { ok: false, code: 'unavailable', message: 'Sign-in is unavailable: try again shortly.' } as const;
const OFFLINE = { ok: false, code: 'offline', message: 'You are offline: check your connection and try again.' } as const;

async function call<T extends object>(action: string, body?: unknown): Promise<AuthResult<T>> {
	return run(() => post<T | undefined>(`/auth/${action}`, body ?? {}));
}

async function run<T extends object>(request: () => Promise<T | undefined>): Promise<AuthResult<T>> {
	try {
		const out = await request();
		return { ok: true, ...(out ?? ({} as T)) };
	} catch (err) {
		if (err instanceof NetworkError) return err.offline ? OFFLINE : UNAVAILABLE;
		if (err instanceof ApiError) {
			const b = (err.body ?? {}) as { code?: unknown; error?: unknown; session?: unknown };
			if (err.status < 500 && typeof b.code === 'string' && typeof b.error === 'string') {
				// `new_password_required` carries the challenge session the next call hands back.
				return typeof b.session === 'string'
					? { ok: false, code: b.code, message: b.error, session: b.session }
					: { ok: false, code: b.code, message: b.error };
			}
			return UNAVAILABLE;
		}
		throw err;
	}
}

/** Signs in. On success the API has set the session cookies. */
export async function signIn(email: string, password: string): Promise<AuthResult> {
	const result = await call('sign-in', { email, password });
	if (result.ok) clearSession();
	return result;
}

/**
 * Finishes the first sign-in of an invited person: they signed in with the temporary password the
 * pool emailed, `signIn` answered `new_password_required` with a `session`, and this trades that
 * session and their own password for the signed-in session cookies.
 */
export async function newPassword(email: string, password: string, session: string): Promise<AuthResult> {
	const result = await call('new-password', { email, password, session });
	if (result.ok) clearSession();
	return result;
}

/** Creates an account. `confirmation_required` is true until the emailed code is confirmed. */
export const signUp = (email: string, password: string) =>
	call<{ confirmation_required: boolean }>('sign-up', { email, password });

/** Confirms an account with the code the pool emailed. */
export const confirm = (email: string, code: string) => call('confirm', { email, code });

/** Emails a new confirmation code. */
export const resend = (email: string) => call('resend', { email });

/** Starts a password reset: emails a code. */
export const forgot = (email: string) => call('forgot', { email });

/** Finishes a password reset with the emailed code. */
export const reset = (email: string, code: string, password: string) =>
	call('reset', { email, code, password });

/** Signs out: the API revokes the refresh token and clears the cookies. */
export async function signOut(): Promise<AuthResult> {
	const result = await call('sign-out');
	clearSession();
	return result;
}

/** How people get an account: `open` (anyone may sign up) or `invite_only` (an administrator adds them). */
export type SignUpMode = 'open' | 'invite_only';

interface AuthConfig {
	mode: SignUpMode;
	/** The group whose members manage people (invite_only); empty when there is none. */
	userAdminGroup: string;
}

let authConfig: Promise<AuthConfig> | null = null;

function getAuthConfig(): Promise<AuthConfig> {
	if (typeof window === 'undefined') return Promise.resolve({ mode: 'open', userAdminGroup: '' });
	authConfig ??= get<{ sign_up?: unknown; user_admin_group?: unknown }>('/auth/config')
		.then((c): AuthConfig => ({
			mode: c?.sign_up === 'invite_only' ? 'invite_only' : 'open',
			userAdminGroup: typeof c?.user_admin_group === 'string' ? c.user_admin_group : ''
		}))
		.catch((): AuthConfig => {
			authConfig = null;
			return { mode: 'open', userAdminGroup: '' };
		});
	return authConfig;
}

/** The group whose members manage people, for screens that offer People only to them. */
export const getUserAdminGroup = async () => (await getAuthConfig()).userAdminGroup;

/**
 * The app's sign-up mode, from the public GET /api/auth/config. It is asked once per page load. A
 * failed or unreadable answer reads as `open` (the API refuses what it does not offer, so the
 * worst case is a link to a screen that says so) and is not remembered. There is no API to ask on
 * the server.
 */
export async function getSignUpMode(): Promise<SignUpMode> {
	return (await getAuthConfig()).mode;
}

/** Forget the remembered mode (tests). */
export function resetSignUpMode(): void {
	authConfig = null;
}

/** Changes the signed-in person's own password. Both modes. */
export const changePassword = (currentPassword: string, newPassword: string) =>
	call('change-password', { current_password: currentPassword, new_password: newPassword });

/** One row of the People list. */
export interface Person {
	email: string;
	status: string;
	enabled: boolean;
	admin: boolean;
	created_at?: string;
}

/** A person's temporary password, shown once: the API never returns it again. */
export interface TemporaryPassword {
	email: string;
	temporary_password: string;
	admin?: boolean;
}

/** The People admin (invite_only, administrators only; anyone else is refused). */
export const listPeople = (next?: string) =>
	run(() => get<{ people: Person[]; next?: string }>('/auth/users', next ? { next } : undefined));

/** Adds a person with a generated temporary password; nothing is emailed. */
export const invitePerson = (email: string, admin: boolean) =>
	call<TemporaryPassword>('users', { email, admin });

export const disablePerson = (email: string) => call('users/disable', { email });

export const enablePerson = (email: string) => call('users/enable', { email });

/** Gives a person a new temporary password, shown once. */
export const resetPersonPassword = (email: string) =>
	call<TemporaryPassword>('users/reset-password', { email });
