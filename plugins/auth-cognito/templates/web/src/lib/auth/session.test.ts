import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError, NetworkError } from '../shell/api';
import { SESSION_TTL_MS, clearSession, getSession } from './session';

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

function stubFetch(impl: (url: string) => Response | Promise<Response>) {
	const fn = vi.fn(async (url: string) => impl(url));
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.stubGlobal('window', {});
	vi.useFakeTimers();
	clearSession();
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

describe('getSession', () => {
	it('asks the API who is signed in', async () => {
		const fetch = stubFetch(() => json(200, { sub: 'u1', email: 'a@example.test', groups: ['admin'] }));
		expect(await getSession()).toEqual({ sub: 'u1', email: 'a@example.test', groups: ['admin'] });
		expect(fetch.mock.calls[0]![0]).toBe('/api/me');
	});

	it('reads a 401 as nobody', async () => {
		stubFetch(() => json(401, { error: 'sign-in required', code: 'unauthenticated' }));
		expect(await getSession()).toBeNull();
	});

	it('reads an answer with no sub as nobody', async () => {
		stubFetch(() => json(200, { hello: 'world' }));
		expect(await getSession()).toBeNull();
	});

	it('throws on any other failure rather than reading it as signed out', async () => {
		stubFetch(() => json(503, { error: 'sign-in is unavailable' }));
		await expect(getSession()).rejects.toBeInstanceOf(ApiError);

		stubFetch(() => {
			throw new TypeError('network down');
		});
		await expect(getSession()).rejects.toBeInstanceOf(NetworkError);
	});

	it('does not remember a failure', async () => {
		let up = false;
		const fetch = stubFetch(() => (up ? json(200, { sub: 'u1' }) : json(503, {})));
		await expect(getSession()).rejects.toBeInstanceOf(ApiError);
		up = true;
		expect(await getSession()).toEqual({ sub: 'u1' });
		expect(fetch).toHaveBeenCalledTimes(2);
	});

	it('asks once for a burst of navigations, and again after the TTL', async () => {
		const fetch = stubFetch(() => json(200, { sub: 'u1' }));
		await Promise.all([getSession(), getSession(), getSession()]);
		await getSession();
		expect(fetch).toHaveBeenCalledTimes(1);

		vi.advanceTimersByTime(SESSION_TTL_MS + 1);
		await getSession();
		expect(fetch).toHaveBeenCalledTimes(2);
	});

	it('forgets the answer when sign-in or sign-out clears it', async () => {
		let who: string | null = null;
		const fetch = stubFetch(() => (who ? json(200, { sub: who }) : json(401, {})));
		expect(await getSession()).toBeNull();

		who = 'u1';
		clearSession();
		expect(await getSession()).toEqual({ sub: 'u1' });
		expect(fetch).toHaveBeenCalledTimes(2);
	});

	it('does not keep an answer that was in flight when the session was cleared', async () => {
		let release: (r: Response) => void = () => {};
		const first = new Promise<Response>((resolve) => (release = resolve));
		const fetch = stubFetch(() => first);
		const stale = getSession();
		clearSession();
		release(json(200, { sub: 'previous-person' }));
		await stale;

		fetch.mockImplementation(async () => json(401, {}));
		expect(await getSession()).toBeNull();
	});

	it('has no session on the server, and does not call the API', async () => {
		vi.unstubAllGlobals();
		const fetch = stubFetch(() => json(200, { sub: 'u1' }));
		expect(await getSession()).toBeNull();
		expect(fetch).not.toHaveBeenCalled();
	});
});
