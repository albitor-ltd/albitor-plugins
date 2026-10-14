import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../shell/api';
import { createQueryClient } from './client';
import { API_KEY, apiQuery } from './query';

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

afterEach(() => vi.unstubAllGlobals());

describe('apiQuery', () => {
	it('keys the query by path and parameters under the shared api prefix', () => {
		const a = apiQuery('/items', { q: 'x' });
		expect(a.queryKey).toEqual([API_KEY, '/items', { q: 'x' }]);
		expect(apiQuery('/items').queryKey).toEqual([API_KEY, '/items', {}]);
		expect(apiQuery('/items', { q: 'y' }).queryKey).not.toEqual(a.queryKey);
	});

	it('fetches through the shell API client and caches the result', async () => {
		const fetch = vi.fn(async () => json(200, { items: [1] }));
		vi.stubGlobal('fetch', fetch);
		const client = createQueryClient();
		const opts = apiQuery<{ items: number[] }>('/items', { page: 2 });
		expect(await client.fetchQuery(opts)).toEqual({ items: [1] });
		expect(await client.fetchQuery(opts)).toEqual({ items: [1] });
		expect(fetch).toHaveBeenCalledTimes(1);
		expect((fetch.mock.calls[0] as unknown as [string])[0]).toBe('/api/items?page=2');
	});

	it('passes the abort signal so a cancelled query cancels the request', async () => {
		let seen: AbortSignal | null | undefined;
		vi.stubGlobal(
			'fetch',
			vi.fn(async (_url: string, init: RequestInit) => {
				seen = init.signal;
				return json(200, {});
			})
		);
		await createQueryClient().fetchQuery(apiQuery('/items'));
		expect(seen).toBeInstanceOf(AbortSignal);
	});

	it('refuses a path that is not app-relative, before any request is made', async () => {
		const fetch = vi.fn();
		vi.stubGlobal('fetch', fetch);
		await expect(createQueryClient().fetchQuery(apiQuery('//evil.example/x'))).rejects.toBeInstanceOf(TypeError);
		await expect(createQueryClient().fetchQuery(apiQuery('/a/../b'))).rejects.toBeInstanceOf(TypeError);
		expect(fetch).not.toHaveBeenCalled();
	});

	it('surfaces a client error without retrying it', async () => {
		const fetch = vi.fn(async () => json(404, { error: 'nope' }));
		vi.stubGlobal('fetch', fetch);
		await expect(createQueryClient().fetchQuery(apiQuery('/items/9'))).rejects.toBeInstanceOf(ApiError);
		expect(fetch).toHaveBeenCalledTimes(1);
	});
});
