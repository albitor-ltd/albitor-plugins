import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { deleteAccount, isHostedUrl, loginUrl, signInErrorMessage, signOut, signOutAndLeave } from './client';

const session = vi.hoisted(() => ({ clearSession: vi.fn() }));
vi.mock('./session', () => session);

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

function stubFetch(impl: (url: string, init: RequestInit) => Response | Promise<Response>) {
	const fn = vi.fn(async (url: string, init: RequestInit) => impl(url, init));
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => session.clearSession.mockReset());
afterEach(() => vi.unstubAllGlobals());

describe('loginUrl', () => {
	it('points at the API, carrying where to land in one encoded parameter', () => {
		expect(loginUrl()).toBe('/api/auth/login');
		const u = new URL(loginUrl('/reports?tab=2&next=//evil.test'), 'https://app.test');
		expect(u.pathname).toBe('/api/auth/login');
		expect(u.searchParams.getAll('next')).toEqual(['/reports?tab=2&next=//evil.test']);
	});
});

describe('signInErrorMessage', () => {
	it('answers a known code, and a general message for an unknown one', () => {
		expect(signInErrorMessage('expired')).toMatch(/took too long/);
		expect(signInErrorMessage('<script>')).toBe(signInErrorMessage('failed'));
	});
});

describe('isHostedUrl', () => {
	it.each([
		['https://pool.auth.eu-west-2.amazoncognito.com/logout?client_id=c', true],
		['http://pool.test/logout', false],
		['javascript:alert(1)', false],
		['/logout', false],
		[undefined, false]
	])('%s is %s', (url, want) => expect(isHostedUrl(url)).toBe(want));
});

describe('signOutAndLeave', () => {
	it('signs out by fetch, then navigates to the Cognito logout page', async () => {
		const fetch = stubFetch(() => json(200, { logout_url: 'https://pool.auth.test/logout?client_id=c' }));
		const assign = vi.fn();
		expect(await signOutAndLeave('/sign-in', assign)).toEqual({ ok: true });
		expect(fetch.mock.calls[0]![0]).toBe('/api/auth/sign-out');
		expect(assign).toHaveBeenCalledExactlyOnceWith('https://pool.auth.test/logout?client_id=c');
	});

	it('goes to the fallback when there is no hosted page (a local run) or the URL is not https', async () => {
		for (const body of [{}, { logout_url: 'javascript:alert(1)' }]) {
			stubFetch(() => json(200, body));
			const assign = vi.fn();
			await signOutAndLeave('/sign-in', assign);
			expect(assign).toHaveBeenCalledExactlyOnceWith('/sign-in');
		}
	});

	it('does not navigate when the sign-out failed', async () => {
		stubFetch(() => json(503, { error: 'x' }));
		const assign = vi.fn();
		expect(await signOutAndLeave('/sign-in', assign)).toMatchObject({ ok: false });
		expect(assign).not.toHaveBeenCalled();
	});
});

describe('signOut', () => {
	it('posts JSON to /api/auth/sign-out, forgets the session and answers the logout URL', async () => {
		const fetch = stubFetch(() => json(200, { logout_url: 'https://pool.test/logout' }));
		expect(await signOut()).toEqual({ ok: true, logout_url: 'https://pool.test/logout' });
		const [url, init] = fetch.mock.calls[0]!;
		expect([url, init.method, init.body]).toEqual(['/api/auth/sign-out', 'POST', '{}']);
		expect(session.clearSession).toHaveBeenCalled();
	});

	it('forgets the session even when the API refused', async () => {
		stubFetch(() => json(501, { error: 'sign-in is not available in a local run', code: 'local' }));
		expect(await signOut()).toMatchObject({ ok: false, code: 'local' });
		expect(session.clearSession).toHaveBeenCalled();
	});
});

describe('deleteAccount', () => {
	it('posts to /api/auth/delete and forgets the session on success', async () => {
		const fetch = stubFetch(() => json(200, { logout_url: 'https://pool.test/logout' }));
		expect(await deleteAccount()).toEqual({ ok: true, logout_url: 'https://pool.test/logout' });
		expect(fetch.mock.calls[0]![0]).toBe('/api/auth/delete');
		expect(session.clearSession).toHaveBeenCalled();
	});

	it('answers a refusal as a result, and keeps the session', async () => {
		stubFetch(() => json(403, { error: 'this app cannot delete accounts yet', code: 'scope_missing' }));
		expect(await deleteAccount()).toEqual({
			ok: false,
			code: 'scope_missing',
			message: 'this app cannot delete accounts yet'
		});
		expect(session.clearSession).not.toHaveBeenCalled();
	});

	it('answers a server fault or an unreadable body as unavailable', async () => {
		stubFetch(() => json(502, { error: 'cognito said something private', code: 'unavailable' }));
		expect(await deleteAccount()).toMatchObject({ ok: false, code: 'unavailable' });
		stubFetch(() => new Response('<html>', { status: 400 }));
		expect(await deleteAccount()).toMatchObject({ ok: false, code: 'unavailable' });
	});
});
