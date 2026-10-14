// The LocalStore contract, run against both implementations, and then the whole engine against
// IndexedDB (the engine tests use the memory store).
import { IDBFactory } from 'fake-indexeddb';
import { describe, expect, it } from 'vitest';
import { createSyncEngine } from './engine';
import { dbName, idbStores } from './idb';
import { memoryStores } from './memory';
import { FakeServer } from './testing';
import type { StoreFactory } from './types';

const impls: Array<[string, () => StoreFactory]> = [
	['memory', () => memoryStores()],
	['indexeddb', () => idbStores('t', new IDBFactory())]
];

describe.each(impls)('LocalStore (%s)', (_name, make) => {
	it('keeps records per collection and returns copies', async () => {
		const s = make()('alice');
		await s.putRecord('todos', { id: 'a', title: 'one' });
		await s.putRecord('notes', { id: 'a', title: 'other collection' });
		const got = await s.getRecord('todos', 'a');
		expect(got).toEqual({ id: 'a', title: 'one' });
		got!.title = 'mutated';
		expect(await s.getRecord('todos', 'a')).toEqual({ id: 'a', title: 'one' });
		expect(await s.listRecords('todos')).toHaveLength(1);
		await s.deleteRecord('todos', 'a');
		expect(await s.getRecord('todos', 'a')).toBeUndefined();
		expect(await s.getRecord('notes', 'a')).toBeDefined();
	});

	it('numbers outbox entries in order and replaces one by id', async () => {
		const s = make()('alice');
		const base = { collection: 'todos', entityId: 'a', op: 'create' as const, rev: 1, state: 'pending' as const, attempts: 0, createdAt: 1 };
		const one = await s.putOutbox(base);
		const two = await s.putOutbox({ ...base, entityId: 'b' });
		expect(two).toBeGreaterThan(one);
		await s.putOutbox({ ...base, id: one, rev: 2 });
		const all = await s.listOutbox();
		expect(all.map((e) => [e.id, e.rev])).toEqual([[one, 2], [two, 1]]);
		await s.deleteOutbox(one);
		expect((await s.listOutbox()).map((e) => e.id)).toEqual([two]);
	});

	it('remembers a cursor per collection and clears everything on request', async () => {
		const s = make()('alice');
		expect(await s.getCursor('todos')).toBeNull();
		await s.setCursor('todos', '7');
		await s.putRecord('todos', { id: 'a' });
		await s.putOutbox({ collection: 'todos', entityId: 'a', op: 'delete', rev: 1, state: 'pending', attempts: 0, createdAt: 1 });
		expect(await s.getCursor('todos')).toBe('7');
		await s.clear();
		expect(await s.getCursor('todos')).toBeNull();
		expect(await s.listRecords('todos')).toEqual([]);
		expect(await s.listOutbox()).toEqual([]);
	});

	it('gives each owner a separate store', async () => {
		const stores = make();
		await stores('alice').putRecord('todos', { id: 'a', title: 'private' });
		expect(await stores('bob').listRecords('todos')).toEqual([]);
		expect(await stores('alice').listRecords('todos')).toHaveLength(1);
	});
});

describe('indexeddb names', () => {
	it('encode the owner so two ids cannot share a database', () => {
		expect(dbName('p', 'a:b')).not.toBe(dbName('p:a', 'b'));
		expect(dbName('p', 'a/b')).toBe('p:a%2Fb');
	});
});

describe('the engine on IndexedDB', () => {
	it('survives a reload: the queue is still there and uploads', async () => {
		const factory = new IDBFactory();
		const server = new FakeServer();
		const boot = () =>
			createSyncEngine({ stores: idbStores('t', factory), transport: server, collections: ['todos'], lock: (_n, fn) => fn() });

		const before = boot();
		await before.attach('alice');
		server.offline = true;
		const todo = await before.create('todos', { title: 'written offline' });
		await before.sync();

		const after = boot(); // a new page load: new engine, same database
		await after.attach('alice');
		expect((await after.status()).pending).toBe(1);
		server.offline = false;
		await after.sync();
		expect(server.records('alice', 'todos')).toEqual([{ id: todo.id, title: 'written offline' }]);
	});
});
