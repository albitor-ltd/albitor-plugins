import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../shell/api';
import { setSyncUser } from './session';
import { apiTransport, parsePull } from './transport';
import type { OutboxEntry } from './types';

const entry = (over: Partial<OutboxEntry>): OutboxEntry => ({
	id: 1,
	collection: 'items',
	entityId: 'a1',
	op: 'create',
	payload: { name: 'x' },
	rev: 1,
	state: 'inflight',
	attempts: 0,
	createdAt: 0,
	...over
});

function stubFetch(status = 200, body: unknown = {}) {
	const calls: Array<{ url: string; method: string; body: unknown }> = [];
	vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
		calls.push({ url, method: init.method ?? 'GET', body: init.body ? JSON.parse(String(init.body)) : undefined });
		return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
	});
	return calls;
}

afterEach(() => {
	vi.unstubAllGlobals();
	setSyncUser(null);
});

describe('apiTransport', () => {
	it('speaks the REST contract for create, update and delete', async () => {
		const calls = stubFetch();
		setSyncUser('alice');
		const t = apiTransport();
		await t.push(entry({}), 'alice');
		await t.push(entry({ op: 'update', payload: { name: 'y' } }), 'alice');
		await t.push(entry({ op: 'delete', payload: undefined }), 'alice');
		expect(calls).toEqual([
			{ url: '/api/items', method: 'POST', body: { name: 'x', id: 'a1' } },
			{ url: '/api/items/a1', method: 'PUT', body: { name: 'y' } },
			{ url: '/api/items/a1', method: 'DELETE', body: undefined }
		]);
	});

	it('refuses to send a queue that belongs to someone other than the signed-in person', async () => {
		const calls = stubFetch();
		setSyncUser('bob');
		const t = apiTransport();
		await expect(t.push(entry({}), 'alice')).rejects.toMatchObject({ status: 401 });
		await expect(t.pull('items', null, 'alice')).rejects.toBeInstanceOf(ApiError);
		expect(calls).toEqual([]);
	});

	it('refuses everything when nobody is signed in', async () => {
		const calls = stubFetch();
		await expect(apiTransport().push(entry({}), 'alice')).rejects.toBeInstanceOf(ApiError);
		expect(calls).toEqual([]);
	});

	it('cannot be steered by a record id into another path', async () => {
		const calls = stubFetch();
		setSyncUser('alice');
		await expect(apiTransport().push(entry({ op: 'delete', entityId: '..' }), 'alice')).rejects.toThrow(TypeError);
		expect(calls).toEqual([]);
	});

	it('sends the cursor on a pull and parses the answer', async () => {
		const calls = stubFetch(200, { items: [{ id: 'a1', name: 'x' }], deleted: ['z'], cursor: '9' });
		setSyncUser('alice');
		const first = await apiTransport().pull('items', null, 'alice');
		await apiTransport().pull('items', '9', 'alice');
		expect(calls.map((c) => c.url)).toEqual(['/api/items', '/api/items?since=9']);
		expect(first).toEqual({ upserts: [{ id: 'a1', name: 'x' }], deleted: ['z'], cursor: '9' });
	});
});

describe('apiTransport timeouts', () => {
	it('abandons a request that hangs, as an unreachable API (so the sync lock is released)', async () => {
		vi.stubGlobal(
			'fetch',
			(_url: string, init: RequestInit) =>
				new Promise((_resolve, reject) => init.signal?.addEventListener('abort', () => reject(init.signal?.reason)))
		);
		setSyncUser('alice');
		await expect(apiTransport({ timeoutMs: 20 }).push(entry({}), 'alice')).rejects.toMatchObject({ name: 'NetworkError' });
	});
});

describe('parsePull', () => {
	it('defaults a missing deleted list and cursor', () => {
		expect(parsePull({ items: [] })).toEqual({ upserts: [], deleted: [], cursor: null });
	});

	it.each([
		['not an object', null],
		['items missing', {}],
		['a record with no id', { items: [{ name: 'x' }] }],
		['a record with an empty id', { items: [{ id: '' }] }],
		['a record id that could steer a path', { items: [{ id: '../x' }] }],
		['a tombstone that could steer a path', { items: [], deleted: ['a/b'] }],
		['a non-string tombstone', { items: [], deleted: [3] }],
		['a numeric cursor', { items: [], cursor: 5 }]
	])('refuses %s', (_name, body) => {
		expect(() => parsePull(body)).toThrow(/contract/);
	});
});
