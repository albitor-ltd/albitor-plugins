import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	ApiError,
	MAX_ATTEMPTS,
	MAX_IN_FLIGHT,
	REQUEST_TIMEOUT_MS,
	NetworkError,
	api,
	get,
	post,
	setTokenSource
} from './api';

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

function stubFetch(impl: (url: string, init: RequestInit) => Response | Promise<Response>) {
	const fn = vi.fn(async (url: string, init: RequestInit) => impl(url, init));
	vi.stubGlobal('fetch', fn);
	return fn;
}

afterEach(() => {
	setTokenSource(null);
	vi.useRealTimers();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});

const throttled = (status = 503, headers: Record<string, string> = {}) =>
	new Response('{"message":"Service Unavailable"}', {
		status,
		headers: { 'Content-Type': 'application/json', ...headers }
	});

/** Fake timers and a Math.random of 1, so a backoff waits its full ceiling (250ms, then 500ms). */
function fakeTime() {
	vi.useFakeTimers();
	vi.spyOn(Math, 'random').mockReturnValue(1);
}

describe('api', () => {
	it('calls /api, encodes the query and skips undefined values', async () => {
		const fetch = stubFetch(() => json(200, { ok: true }));
		const out = await get<{ ok: boolean }>('/items', { q: 'a b', page: 2, skip: undefined });
		expect(out).toEqual({ ok: true });
		expect(fetch.mock.calls[0]![0]).toBe('/api/items?q=a+b&page=2');
		expect(fetch.mock.calls[0]![1].method).toBe('GET');
	});

	it('sends a JSON body as POST and the bearer token when a source is set', async () => {
		const fetch = stubFetch(() => json(201, { id: '1' }));
		setTokenSource(async () => 'tok-1');
		await post('/items', { name: 'x' });
		const init = fetch.mock.calls[0]![1];
		expect(init.method).toBe('POST');
		expect(init.body).toBe('{"name":"x"}');
		expect(init.headers).toMatchObject({
			Authorization: 'Bearer tok-1',
			'Content-Type': 'application/json'
		});
	});

	it('sends no Authorization header without a token', async () => {
		const fetch = stubFetch(() => json(200, {}));
		setTokenSource(() => null);
		await get('/items');
		expect(fetch.mock.calls[0]![1].headers).not.toHaveProperty('Authorization');
	});

	it('returns undefined for 204', async () => {
		stubFetch(() => new Response(null, { status: 204 }));
		expect(await api('/items/1', { method: 'DELETE' })).toBeUndefined();
	});

	it('throws ApiError carrying the status and parsed body', async () => {
		stubFetch(() => json(409, { error: 'exists' }));
		await expect(post('/items', {})).rejects.toMatchObject({
			name: 'ApiError',
			status: 409,
			body: { error: 'exists' }
		});
		await expect(post('/items', {})).rejects.toBeInstanceOf(ApiError);
	});

	it('throws NetworkError, flagged offline when the browser says so', async () => {
		stubFetch(() => {
			throw new TypeError('failed to fetch');
		});
		vi.stubGlobal('navigator', { onLine: false });
		await expect(get('/items')).rejects.toMatchObject({ offline: true });
		vi.stubGlobal('navigator', { onLine: true });
		const err = await get('/items').catch((e: unknown) => e);
		expect(err).toBeInstanceOf(NetworkError);
		expect((err as NetworkError).offline).toBe(false);
	});

	it.each([
		'items',
		'//evil.test/x',
		'https://evil.test/x',
		'/items?x=1',
		'/items#a',
		'/a/../b',
		'/./x',
		'/%2e%2e/admin',
		'/%2E%2e/admin',
		'/a/%2E%2E',
		'/a%2fb',
		'/a%2Fb',
		'/a%5Cb',
		'/a%5cb',
		'/a b',
		''
	])('refuses the path %j before any request', async (path) => {
		const fetch = stubFetch(() => json(200, {}));
		await expect(get(path)).rejects.toBeInstanceOf(TypeError);
		expect(fetch).not.toHaveBeenCalled();
	});

	describe('retry', () => {
		it('retries a 503 on a GET and then succeeds', async () => {
			fakeTime();
			let n = 0;
			const fetch = stubFetch(() => (++n < 3 ? throttled() : json(200, { ok: true })));
			const p = get('/items');
			await vi.advanceTimersByTimeAsync(250);
			expect(fetch).toHaveBeenCalledTimes(2);
			await vi.advanceTimersByTimeAsync(500);
			await expect(p).resolves.toEqual({ ok: true });
			expect(fetch).toHaveBeenCalledTimes(3);
		});

		it('retries a 429 and an unreachable API on a GET, but not an offline browser', async () => {
			fakeTime();
			let n = 0;
			stubFetch(() => {
				if (++n === 1) return throttled(429);
				if (n === 2) throw new TypeError('failed to fetch');
				return json(200, {});
			});
			vi.stubGlobal('navigator', { onLine: true });
			const p = get('/items');
			await vi.advanceTimersByTimeAsync(1000);
			await expect(p).resolves.toEqual({});

			const fetch = stubFetch(() => {
				throw new TypeError('failed to fetch');
			});
			vi.stubGlobal('navigator', { onLine: false });
			await expect(get('/items')).rejects.toMatchObject({ offline: true, retried: false });
			expect(fetch).toHaveBeenCalledTimes(1);
		});

		it('retries a HEAD', async () => {
			fakeTime();
			let n = 0;
			const fetch = stubFetch(() => (++n < 2 ? throttled() : new Response(null, { status: 200 })));
			const p = api('/items', { method: 'HEAD' });
			await vi.advanceTimersByTimeAsync(250);
			await p;
			expect(fetch).toHaveBeenCalledTimes(2);
		});

		it('never waits less than Retry-After seconds', async () => {
			fakeTime();
			vi.spyOn(Math, 'random').mockReturnValue(0);
			let n = 0;
			const fetch = stubFetch(() => (++n < 2 ? throttled(503, { 'Retry-After': '3' }) : json(200, {})));
			const p = get('/items');
			await vi.advanceTimersByTimeAsync(2999);
			expect(fetch).toHaveBeenCalledTimes(1);
			await vi.advanceTimersByTimeAsync(1);
			await p;
			expect(fetch).toHaveBeenCalledTimes(2);
		});

		it('reads Retry-After as an HTTP date', async () => {
			fakeTime();
			vi.setSystemTime(new Date('2030-01-01T00:00:00Z'));
			vi.spyOn(Math, 'random').mockReturnValue(0);
			let n = 0;
			const when = new Date('2030-01-01T00:00:02Z').toUTCString();
			const fetch = stubFetch(() => (++n < 2 ? throttled(429, { 'Retry-After': when }) : json(200, {})));
			const p = get('/items');
			await vi.advanceTimersByTimeAsync(1999);
			expect(fetch).toHaveBeenCalledTimes(1);
			await vi.advanceTimersByTimeAsync(1);
			await p;
			expect(fetch).toHaveBeenCalledTimes(2);
		});

		it('fails at once, marked retried, when Retry-After exceeds the cap', async () => {
			fakeTime();
			const fetch = stubFetch(() => throttled(429, { 'Retry-After': '6' }));
			await expect(get('/items')).rejects.toMatchObject({ status: 429, retried: true });
			expect(fetch).toHaveBeenCalledTimes(1);
		});

		it('gives up after the attempt limit and marks the error retried', async () => {
			fakeTime();
			const fetch = stubFetch(() => throttled());
			const p = get('/items').catch((e: unknown) => e);
			await vi.advanceTimersByTimeAsync(10_000);
			const err = await p;
			expect(err).toBeInstanceOf(ApiError);
			expect(err).toMatchObject({ status: 503, retried: true });
			expect(fetch).toHaveBeenCalledTimes(MAX_ATTEMPTS);
		});

		it('never retries a write, even when throttled', async () => {
			for (const method of ['POST', 'PUT', 'PATCH', 'DELETE'] as const) {
				const fetch = stubFetch(() => throttled());
				await expect(api('/items', { method })).rejects.toMatchObject({ status: 503, retried: false });
				expect(fetch).toHaveBeenCalledTimes(1);
			}
			const fetch = stubFetch(() => {
				throw new TypeError('failed to fetch');
			});
			await expect(post('/items', {})).rejects.toBeInstanceOf(NetworkError);
			expect(fetch).toHaveBeenCalledTimes(1);
		});

		it('does not retry a 404 or a 500', async () => {
			for (const status of [404, 500]) {
				const fetch = stubFetch(() => json(status, {}));
				await expect(get('/items')).rejects.toMatchObject({ status, retried: false });
				expect(fetch).toHaveBeenCalledTimes(1);
			}
		});

		it('reads the token again on each attempt', async () => {
			fakeTime();
			let t = 0;
			setTokenSource(() => `tok-${++t}`);
			let n = 0;
			const fetch = stubFetch(() => (++n < 2 ? throttled() : json(200, {})));
			const p = get('/items');
			await vi.advanceTimersByTimeAsync(250);
			await p;
			expect(fetch.mock.calls[0]![1].headers).toMatchObject({ Authorization: 'Bearer tok-1' });
			expect(fetch.mock.calls[1]![1].headers).toMatchObject({ Authorization: 'Bearer tok-2' });
		});

		it('refuses a GET or HEAD with a body before any request', async () => {
			const fetch = stubFetch(() => json(200, {}));
			for (const method of ['GET', 'HEAD'] as const) {
				await expect(api('/items', { method, body: { a: 1 } })).rejects.toBeInstanceOf(TypeError);
			}
			expect(fetch).not.toHaveBeenCalled();
		});

		it('rejects with AbortError when the caller aborts a request in flight', async () => {
			const fetch = vi.fn(
				(_url: string, init: RequestInit) =>
					new Promise<Response>((_resolve, reject) =>
						init.signal!.addEventListener('abort', () =>
							reject(new DOMException('aborted', 'AbortError'))
						)
					)
			);
			vi.stubGlobal('fetch', fetch);
			const ctl = new AbortController();
			const p = api('/items', { signal: ctl.signal });
			const assertion = expect(p).rejects.toMatchObject({ name: 'AbortError' });
			await Promise.resolve();
			ctl.abort();
			await assertion;
			expect(fetch).toHaveBeenCalledTimes(1);
		});

		it('aborts immediately during the backoff wait', async () => {
			fakeTime();
			const fetch = stubFetch(() => throttled());
			const ctl = new AbortController();
			const p = api('/items', { signal: ctl.signal });
			const assertion = expect(p).rejects.toMatchObject({ name: 'AbortError' });
			await vi.advanceTimersByTimeAsync(10);
			expect(fetch).toHaveBeenCalledTimes(1);
			ctl.abort();
			await assertion;
			await vi.advanceTimersByTimeAsync(10_000);
			expect(fetch).toHaveBeenCalledTimes(1);
		});
	});

	describe('in-flight cap', () => {
		/** Fetches that stay open until released, so a test controls how many are in flight. */
		function gated() {
			const open: Array<() => void> = [];
			const fetch = stubFetch(
				() => new Promise<Response>((resolve) => open.push(() => resolve(json(200, {}))))
			);
			return { fetch, open };
		}
		const tick = () => new Promise((r) => setTimeout(r, 0));

		it('holds the request past the cap until one finishes, in order', async () => {
			const { fetch, open } = gated();
			const calls = Array.from({ length: MAX_IN_FLIGHT + 2 }, (_, i) => get(`/r${i}`));
			await tick();
			expect(fetch).toHaveBeenCalledTimes(MAX_IN_FLIGHT);
			open[0]!();
			await tick();
			expect(fetch).toHaveBeenCalledTimes(MAX_IN_FLIGHT + 1);
			expect(fetch.mock.calls[MAX_IN_FLIGHT]![0]).toBe(`/api/r${MAX_IN_FLIGHT}`);
			open[1]!();
			await tick();
			expect(fetch.mock.calls[MAX_IN_FLIGHT + 1]![0]).toBe(`/api/r${MAX_IN_FLIGHT + 1}`);
			for (const o of open) o();
			await tick();
			for (const o of open) o();
			await Promise.all(calls);
		});

		it('frees the slot when a request fails', async () => {
			let fail = true;
			const fetch = stubFetch(() => (fail ? json(404, {}) : json(200, {})));
			for (let i = 0; i < MAX_IN_FLIGHT + 2; i++) {
				await expect(get('/x')).rejects.toBeInstanceOf(ApiError);
			}
			fail = false;
			await get('/x');
			expect(fetch).toHaveBeenCalledTimes(MAX_IN_FLIGHT + 3);
		});

		it('does not hold a slot while waiting in backoff', async () => {
			fakeTime();
			stubFetch((url) => (url === '/api/t' ? throttled() : json(200, { url })));
			const throttledReads = Array.from({ length: MAX_IN_FLIGHT }, () =>
				get('/t').catch(() => undefined)
			);
			// Every slot has been throttled and is now waiting out its backoff.
			await vi.advanceTimersByTimeAsync(10);
			const other = get('/other');
			await vi.advanceTimersByTimeAsync(10);
			await expect(other).resolves.toEqual({ url: '/api/other' });
			await vi.advanceTimersByTimeAsync(20_000);
			await Promise.all(throttledReads);
		});

		it('drops a queued request whose signal aborts, without consuming a slot', async () => {
			const { fetch, open } = gated();
			const first = Array.from({ length: MAX_IN_FLIGHT }, (_, i) => get(`/r${i}`));
			await tick();
			const ctl = new AbortController();
			const queued = api('/queued', { signal: ctl.signal });
			const after = get('/after');
			await tick();
			expect(fetch).toHaveBeenCalledTimes(MAX_IN_FLIGHT);
			ctl.abort();
			await expect(queued).rejects.toMatchObject({ name: 'AbortError' });
			open[0]!();
			await tick();
			// The slot went to the next in line, not to the aborted request.
			expect(fetch.mock.calls[MAX_IN_FLIGHT]![0]).toBe('/api/after');
			expect(fetch.mock.calls.some((c) => c[0] === '/api/queued')).toBe(false);
			for (const o of open) o();
			await Promise.all([...first, after]);
		});

		it('frees the slot of a hung request after the timeout, then retries the read', async () => {
			fakeTime();
			let hung = 0;
			const fetch = vi.fn(
				(url: string, init: RequestInit) =>
					new Promise<Response>((resolve, reject) => {
						if (url === '/api/ok') return resolve(json(200, { ok: true }));
						hung++;
						init.signal!.addEventListener('abort', () =>
							reject(new DOMException('aborted', 'AbortError'))
						);
					})
			);
			vi.stubGlobal('fetch', fetch);
			const hungReads = Array.from({ length: MAX_IN_FLIGHT }, () =>
				get('/hang').catch((e: unknown) => e)
			);
			const queued = get('/ok');
			await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS - 1);
			expect(hung).toBe(MAX_IN_FLIGHT);
			expect(fetch).toHaveBeenCalledTimes(MAX_IN_FLIGHT);
			await vi.advanceTimersByTimeAsync(1);
			// The timeouts freed the slots, so the queued request ran.
			await expect(queued).resolves.toEqual({ ok: true });
			await vi.advanceTimersByTimeAsync(10 * REQUEST_TIMEOUT_MS);
			const errs = await Promise.all(hungReads);
			for (const e of errs) expect(e).toMatchObject({ name: 'NetworkError', offline: false, retried: true });
		});

		it('rejects at once for a signal that is already aborted', async () => {
			const fetch = stubFetch(() => json(200, {}));
			await expect(api('/x', { signal: AbortSignal.abort() })).rejects.toMatchObject({
				name: 'AbortError'
			});
			expect(fetch).not.toHaveBeenCalled();
		});
	});
});
