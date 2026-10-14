import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, NetworkError, api, get, post, setTokenSource } from './api';

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

function stubFetch(impl: (url: string, init: RequestInit) => Response | Promise<Response>) {
	const fn = vi.fn(async (url: string, init: RequestInit) => impl(url, init));
	vi.stubGlobal('fetch', fn);
	return fn;
}

afterEach(() => {
	setTokenSource(null);
	vi.unstubAllGlobals();
});

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
		'/a b',
		''
	])('refuses the path %j before any request', async (path) => {
		const fetch = stubFetch(() => json(200, {}));
		await expect(get(path)).rejects.toBeInstanceOf(TypeError);
		expect(fetch).not.toHaveBeenCalled();
	});
});
