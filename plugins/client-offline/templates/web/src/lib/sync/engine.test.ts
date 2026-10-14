import { describe, expect, it } from 'vitest';
import { ApiError } from '../shell/api';
import { createSyncEngine, type SyncEngine } from './engine';
import { memoryStores } from './memory';
import { FakeServer } from './testing';
import type { StoreFactory } from './types';

const deferred = () => {
	let resolve!: () => void;
	const promise = new Promise<void>((r) => (resolve = r));
	return { promise, resolve };
};

/** A device: its own local stores, talking to a shared server. */
function device(server: FakeServer, stores: StoreFactory = memoryStores()): SyncEngine {
	return createSyncEngine({ stores, transport: server, collections: ['todos'], lock: (_n, fn) => fn() });
}

const titles = async (e: SyncEngine) => (await e.list('todos')).map((r) => r.title).sort();

describe('an edit made while its create is in flight', () => {
	it('is not lost when the create completes', async () => {
		const server = new FakeServer();
		const gate = deferred();
		server.gate = gate.promise;
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'draft' });

		const run = a.flush(); // the create is now on the wire, held by the gate
		await new Promise((r) => setTimeout(r, 0));
		await a.update('todos', todo.id, { title: 'final' });
		gate.resolve();
		await run;
		await a.flush();

		expect(server.records('alice', 'todos')).toEqual([{ id: todo.id, title: 'final' }]);
		expect((await a.status()).pending).toBe(0);
	});

	it('amends a create that has not been sent yet, so one request goes out', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'draft' });
		await a.update('todos', todo.id, { title: 'better' });
		await a.update('todos', todo.id, { done: true });
		await a.flush();
		expect(server.pushes).toHaveLength(1);
		expect(server.records('alice', 'todos')).toEqual([{ id: todo.id, title: 'better', done: true }]);
	});
});

describe('a different person signing in on the same device', () => {
	it('sees none of the previous person data and uploads none of their queue', async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const a = device(server, stores);
		await a.attach('alice');
		server.offline = true;
		await a.create('todos', { title: "alice's unsynced note" });
		a.detach(); // the session lapsed

		server.offline = false;
		await a.attach('bob');
		expect(await a.list('todos')).toEqual([]);
		await a.sync();

		expect(server.pushes.filter((p) => p.owner === 'bob')).toEqual([]);
		expect(server.records('bob', 'todos')).toEqual([]);
		expect((await a.status()).pending).toBe(0);
	});

	it("keeps the previous person's queue for when they sign back in", async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const a = device(server, stores);
		await a.attach('alice');
		server.offline = true;
		await a.create('todos', { title: 'kept' });
		a.detach();
		await a.attach('bob');
		a.detach();

		server.offline = false;
		await a.attach('alice');
		expect(await titles(a)).toEqual(['kept']);
		await a.flush();
		expect(server.records('alice', 'todos').map((r) => r.title)).toEqual(['kept']);
	});

	it('stops a flush that was running when the person changed', async () => {
		const server = new FakeServer();
		const gate = deferred();
		const a = device(server);
		await a.attach('alice');
		await a.create('todos', { title: 'one' });
		await a.create('todos', { title: 'two' });
		server.gate = gate.promise;
		const run = a.flush();
		await new Promise((r) => setTimeout(r, 0));
		a.detach();
		await a.attach('bob');
		server.gate = null;
		gate.resolve();
		const result = await run;
		expect(result.stoppedBy).toBe('detached');
		expect(server.pushes.every((p) => p.owner === 'alice')).toBe(true);
		expect(server.pushes.length).toBeLessThanOrEqual(1);
	});
});

describe('deletions', () => {
	it('reach other devices', async () => {
		const server = new FakeServer();
		const phone = device(server);
		const laptop = device(server);
		await phone.attach('alice');
		await laptop.attach('alice');
		const todo = await phone.create('todos', { title: 'to go' });
		await phone.sync();
		await laptop.sync();
		expect(await titles(laptop)).toEqual(['to go']);

		await phone.remove('todos', todo.id);
		await phone.sync();
		await laptop.sync();
		expect(await laptop.list('todos')).toEqual([]);
	});

	it('of a record never uploaded cancel the create instead of sending anything', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'oops' });
		await a.remove('todos', todo.id);
		await a.flush();
		expect(server.pushes).toEqual([]);
		expect((await a.status()).pending).toBe(0);
	});

	it('made offline are sent when the device is back', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'x' });
		await a.sync();
		server.offline = true;
		await a.remove('todos', todo.id);
		expect((await a.sync()).stoppedBy).toBe('offline');
		server.offline = false;
		await a.sync();
		expect(server.records('alice', 'todos')).toEqual([]);
	});

	it('pulled from the server drop the record but not unsent local work', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.seed('alice', 'todos', { id: 'r1', title: 'remote' });
		await a.sync();
		server.removeRemote('alice', 'todos', 'r1');
		const mine = await a.create('todos', { title: 'mine' });
		server.offline = true;
		await a.update('todos', mine.id, { title: 'mine, edited' });
		server.offline = false;
		await a.sync();
		expect(await titles(a)).toEqual(['mine, edited']);
	});
});

describe('signing out', () => {
	it('refuses while work is unsynced, rather than discarding it', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.offline = true;
		await a.create('todos', { title: 'unsynced' });
		const result = await a.signOut();
		expect(result).toEqual({ ok: false, pending: 1 });
		expect(a.owner).toBe('alice');
		expect(await titles(a)).toEqual(['unsynced']);
	});

	it('uploads what it can first and then signs out', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		await a.create('todos', { title: 'fine' });
		expect(await a.signOut()).toEqual({ ok: true });
		expect(a.owner).toBeNull();
		expect(server.records('alice', 'todos')).toHaveLength(1);
	});

	it('discards the queue only when asked to', async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const a = device(server, stores);
		await a.attach('alice');
		server.offline = true;
		await a.create('todos', { title: 'gone' });
		expect(await a.signOut({ discard: true })).toEqual({ ok: true });
		await a.attach('alice');
		expect(await a.list('todos')).toEqual([]);
		expect((await a.status()).pending).toBe(0);
	});

	it('keeps the work when the session simply detaches', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.offline = true;
		await a.create('todos', { title: 'still here' });
		a.detach();
		await a.attach('alice');
		expect(await titles(a)).toEqual(['still here']);
		expect((await a.status()).pending).toBe(1);
	});
});

describe('concurrent flushes', () => {
	it('share one run: the guard holds however many callers ask', async () => {
		const server = new FakeServer();
		const gate = deferred();
		server.gate = gate.promise;
		const a = device(server);
		await a.attach('alice');
		for (const title of ['a', 'b', 'c']) await a.create('todos', { title });

		const runs = [a.flush(), a.flush(), a.sync(), a.flush(), a.sync()];
		await new Promise((r) => setTimeout(r, 0));
		gate.resolve();
		await Promise.all(runs);

		expect(server.maxActive).toBe(1);
		expect(server.pushes).toHaveLength(3);
	});

	it('picks up work added during a run without starting a second one', async () => {
		const server = new FakeServer();
		const gate = deferred();
		server.gate = gate.promise;
		const a = device(server);
		await a.attach('alice');
		await a.create('todos', { title: 'first' });
		const run = a.flush();
		await new Promise((r) => setTimeout(r, 0));
		await a.create('todos', { title: 'second' });
		const again = a.flush();
		gate.resolve();
		await Promise.all([run, again]);
		expect(server.maxActive).toBe(1);
		expect(server.records('alice', 'todos')).toHaveLength(2);
	});

	it('takes the cross-tab lock for the whole run', async () => {
		const names: string[] = [];
		const server = new FakeServer();
		const a = createSyncEngine({
			stores: memoryStores(),
			transport: server,
			collections: ['todos'],
			lock: (name, fn) => (names.push(name), fn())
		});
		await a.attach('alice');
		await a.create('todos', { title: 'x' });
		await Promise.all([a.flush(), a.flush()]);
		expect(names).toEqual(['albitor-sync:alice']);
	});
});

describe('failures', () => {
	it('keeps the queue when the device is offline and retries later', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.offline = true;
		await a.create('todos', { title: 'later' });
		expect((await a.flush()).stoppedBy).toBe('offline');
		expect((await a.status()).pending).toBe(1);
		server.offline = false;
		await a.flush();
		expect(server.records('alice', 'todos')).toHaveLength(1);
	});

	it('pauses on a lapsed session and flags it, losing nothing', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		await a.create('todos', { title: 'x' });
		server.failNext = new ApiError(401, null);
		expect((await a.flush()).stoppedBy).toBe('auth');
		const s = await a.status();
		expect(s.needsSignIn).toBe(true);
		expect(s.pending).toBe(1);
		await a.flush();
		expect((await a.status()).needsSignIn).toBe(false);
	});

	it('marks a rejected change failed, keeps it visible and carries on with the rest', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		await a.create('todos', { title: 'bad' });
		await a.create('todos', { title: 'good' });
		server.failNext = new ApiError(422, { secret: 'server internals' });
		const result = await a.flush();
		expect(result.failed).toBe(1);
		const s = await a.status();
		expect(s.failed).toBe(1);
		expect(server.records('alice', 'todos').map((r) => r.title)).toEqual(['good']);
		expect(JSON.stringify(await a.failures())).not.toContain('server internals');
	});

	it('treats a replayed create the server already has as done', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'x' });
		server.seed('alice', 'todos', { id: todo.id, title: 'x' });
		await a.flush();
		expect((await a.status()).pending).toBe(0);
		expect((await a.status()).failed).toBe(0);
	});

	it('puts entries left in flight by a crash back in the queue', async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const a = device(server, stores);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'x' });
		const store = stores('alice');
		const [entry] = await store.listOutbox();
		await store.putOutbox({ ...entry, state: 'inflight' });
		a.detach();
		await a.attach('alice');
		await a.flush();
		expect(server.records('alice', 'todos')).toEqual([{ id: todo.id, title: 'x' }]);
	});
});

describe('pull', () => {
	it('keeps unsent local edits over what the server has', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.seed('alice', 'todos', { id: 'r1', title: 'remote' });
		await a.sync();
		server.offline = true;
		await a.update('todos', 'r1', { title: 'local' });
		server.offline = false;
		server.seed('alice', 'todos', { id: 'r1', title: 'remote 2' });
		await a.sync();
		expect(await titles(a)).toEqual(['local']);
		expect(server.records('alice', 'todos')[0].title).toBe('local');
	});

	it('fetches only what changed since the last pull', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.seed('alice', 'todos', { id: 'r1', title: 'one' });
		await a.sync();
		const seen: Array<string | null> = [];
		const real = server.pull.bind(server);
		server.pull = (c, cursor, o) => (seen.push(cursor), real(c, cursor, o));
		await a.sync();
		expect(seen[0]).not.toBeNull();
	});
});

describe('operations need a signed-in person', () => {
	it('refuses a write with nobody attached', async () => {
		const a = device(new FakeServer());
		await expect(a.create('todos', { title: 'x' })).rejects.toThrow(/not signed in/);
		await expect(a.list('todos')).rejects.toThrow(/not signed in/);
		await expect(a.flush()).rejects.toThrow(/not signed in/);
	});

	it('refuses an unknown collection', async () => {
		const a = device(new FakeServer());
		await a.attach('alice');
		await expect(a.create('nope', { title: 'x' })).rejects.toThrow(/unknown collection/);
	});
});

describe('attach', () => {
	it('lets the last person win when two attaches overlap', async () => {
		const a = device(new FakeServer());
		const first = a.attach('alice');
		const second = a.attach('bob');
		await Promise.all([first, second]);
		expect(a.owner).toBe('bob');
	});

	it('is undone by a detach that arrives while the store opens', async () => {
		const a = device(new FakeServer());
		const pending = a.attach('alice');
		a.detach();
		await pending;
		expect(a.owner).toBeNull();
	});
});

describe('change notifications', () => {
	it('say write for edits and status for sync state, so a listener can avoid looping', async () => {
		const a = device(new FakeServer());
		await a.attach('alice');
		const kinds: string[] = [];
		a.subscribe((k) => kinds.push(k));
		await a.create('todos', { title: 'x' });
		expect(kinds).toEqual(['write']);
		kinds.length = 0;
		await a.sync();
		expect(kinds.length).toBeGreaterThan(0);
		expect(kinds.every((k) => k === 'status')).toBe(true);
	});
});

describe('failed changes', () => {
	it('can be retried or discarded, and discarding a create removes its record', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'bad' });
		server.failNext = new ApiError(422, null);
		await a.flush();
		const [failure] = await a.failures();
		expect(failure.error).toBe('The server rejected this change (422).');

		await a.retryFailed();
		await a.flush();
		expect(server.records('alice', 'todos')).toHaveLength(1);

		const second = await a.create('todos', { title: 'worse' });
		server.failNext = new ApiError(400, null);
		await a.flush();
		const [bad] = await a.failures();
		await a.discardFailed(bad.id);
		expect(await a.get('todos', second.id)).toBeUndefined();
		expect(await a.get('todos', todo.id)).toBeDefined();
		expect((await a.status()).failed).toBe(0);
	});

	it('block later changes to the same record until resolved', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'x' });
		server.failNext = new ApiError(422, null);
		await a.flush();
		await a.update('todos', todo.id, { title: 'y' });
		await a.flush();
		expect(server.pushes).toEqual([]);
	});
});

describe('record ids', () => {
	it('are limited to a plain alphabet because they end up in URLs and keys', async () => {
		const a = device(new FakeServer());
		await a.attach('alice');
		for (const bad of ['../x', 'a/b', 'a b', 'x'.repeat(65)]) {
			await expect(a.create('todos', { id: bad, title: 'x' })).rejects.toThrow(/letters, digits/);
		}
		await expect(a.create('todos', { id: 'ok_1-2', title: 'x' })).resolves.toMatchObject({ id: 'ok_1-2' });
	});

	it('mark a change failed when the API client refuses its request, instead of retrying forever', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		await a.create('todos', { title: 'x' });
		server.failNext = new TypeError('bad path');
		await a.flush();
		expect((await a.status()).failed).toBe(1);
	});
});

describe('review fixes: signing out cannot lose a write that lands during it', () => {
	it('rejects the late write instead of wiping it and reporting success', async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const inner = stores('alice');
		const a = device(server, (owner) => (owner === 'alice' ? wrapped : stores(owner)));
		let calls = 0;
		let late: Promise<unknown> = Promise.resolve();
		const wrapped = {
			...inner,
			listOutbox: async () => {
				const rows = await inner.listOutbox();
				if (++calls === 2) late = a.create('todos', { title: 'late' }).catch((e: Error) => e);
				return rows;
			}
		};
		await a.attach('alice');
		calls = 0; // the first listOutbox of the run is the flush; the second is sign-out counting
		const result = await a.signOut();
		const outcome = await late;
		// Either the write was refused, or it was kept and sign-out declined: never ok with it gone.
		const kept = result.ok === false;
		const refused = outcome instanceof Error && /signing out/.test(outcome.message);
		expect(kept || refused).toBe(true);
		if (result.ok) expect(a.owner).toBeNull();
	});
});

describe('review fixes: a failing entry cannot block the queue forever', () => {
	it('lets later records through, backs off, and marks the entry failed after the cap', async () => {
		let clock = 1_000_000;
		const server = new FakeServer();
		const a = createSyncEngine({ stores: memoryStores(), transport: server, collections: ['todos'], lock: (_n, fn) => fn(), now: () => clock });
		await a.attach('alice');
		const p = await a.create('todos', { title: 'p' });
		await a.create('todos', { title: 'q' });
		server.failWhen = (e) => (e.entityId === p.id ? new ApiError(500, null) : null);
		let attempts = 0;
		const real = server.push.bind(server);
		server.push = (e, o) => (e.entityId === p.id && attempts++, real(e, o));

		await a.sync();
		expect(server.records('alice', 'todos').map((r) => r.title)).toEqual(['q']);
		await a.sync(); // immediately: still backing off
		expect(attempts).toBe(1);
		for (let i = 0; i < 12; i++) {
			clock += 10 * 60_000;
			await a.sync();
		}
		const s = await a.status();
		expect(s.failed).toBe(1);
		expect(s.pending).toBe(0);
		expect(attempts).toBeLessThanOrEqual(6);
		expect((await a.failures())[0].error).toMatch(/kept failing/);
	});
});

describe('review fixes: a tombstone for a record with queued changes is not lost', () => {
	it('is delivered once the queued change is discarded', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.seed('alice', 'todos', { id: 't1', title: 'one' });
		await a.sync();
		await a.update('todos', 't1', { title: 'edited' });
		server.failNext = new ApiError(422, null);
		await a.sync();
		server.removeRemote('alice', 'todos', 't1');
		await a.sync();
		expect(await titles(a)).toEqual(['edited']); // the person's change is not silently dropped
		const [failure] = await a.failures();
		await a.discardFailed(failure.id);
		await a.sync();
		expect(await a.list('todos')).toEqual([]);
	});

	it('keeps local edits over the server copy while they are queued', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.seed('alice', 'todos', { id: 'r1', title: 'remote' });
		await a.sync();
		await a.update('todos', 'r1', { title: 'local' });
		server.failNext = new ApiError(422, null); // the push fails, so this run goes on to pull
		server.seed('alice', 'todos', { id: 'r1', title: 'remote 2' });
		await a.sync();
		expect(await titles(a)).toEqual(['local']);
	});
});

describe('review fixes: mutants that survived', () => {
	it('an entry changed while in flight is sent again, not deleted as if delivered', async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const a = device(server, stores);
		await a.attach('alice');
		await a.create('todos', { title: 'v1' });
		const sent: unknown[] = [];
		const real = server.push.bind(server);
		let first = true;
		server.push = async (e, o) => {
			sent.push(e.payload?.title);
			if (first) {
				first = false;
				const store = stores('alice');
				const [row] = await store.listOutbox();
				await store.putOutbox({ ...row, rev: row.rev + 1, payload: { title: 'v2' } });
				return;
			}
			return real(e, o);
		};
		await a.flush();
		expect(sent).toEqual(['v1', 'v2']);
		expect(server.records('alice', 'todos')[0].title).toBe('v2');
	});

	it('concurrent edits to a record both land in its queued entry', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'x' });
		await Promise.all([a.update('todos', todo.id, { a: 1 }), a.update('todos', todo.id, { b: 2 })]);
		await a.flush();
		expect(server.records('alice', 'todos')).toEqual([{ id: todo.id, title: 'x', a: 1, b: 2 }]);
	});

	it('remove refuses a malformed id up front', async () => {
		const a = device(new FakeServer());
		await a.attach('alice');
		await expect(a.remove('todos', '../x')).rejects.toThrow(/letters, digits/);
	});
});

describe('round 2: sign-out races', () => {
	it('a discard during an in-flight push leaves nothing behind in the cleared store', async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const a = device(server, stores);
		await a.attach('alice');
		await a.create('todos', { title: 'secret' });
		const gate = deferred();
		server.gate = gate.promise;
		server.failWhen = () => new ApiError(400, null);
		const run = a.flush();
		await new Promise((r) => setTimeout(r, 0));
		const out = a.signOut({ discard: true });
		await new Promise((r) => setTimeout(r, 0));
		gate.resolve();
		expect(await out).toEqual({ ok: true });
		await run;
		expect(await stores('alice').listOutbox()).toEqual([]);
		expect(await stores('alice').listRecords('todos')).toEqual([]);
	});

	it('recovers when clearing the store throws: later writes still work', async () => {
		const server = new FakeServer();
		const inner = memoryStores()('alice');
		let failClear = true;
		const store = { ...inner, clear: async () => { if (failClear) throw new Error('disk'); return inner.clear(); } };
		const a = device(server, () => store);
		await a.attach('alice');
		await expect(a.signOut({ discard: true })).rejects.toThrow('disk');
		failClear = false;
		await expect(a.create('todos', { title: 'still works' })).resolves.toBeDefined();
	});

	it('does not sign out the next person who attached while it was finishing', async () => {
		const server = new FakeServer();
		const stores = memoryStores();
		const inner = stores('alice');
		const gate = deferred();
		const slow = { ...inner, clear: async () => { await gate.promise; return inner.clear(); } };
		const a = device(server, (o) => (o === 'alice' ? slow : stores(o)));
		await a.attach('alice');
		const out = a.signOut({ discard: true });
		await new Promise((r) => setTimeout(r, 0));
		await a.attach('bob');
		gate.resolve();
		await out;
		expect(a.owner).toBe('bob');
	});

	it.each(['update', 'remove'] as const)('refuses a late %s instead of letting it be wiped', async (kind) => {
		const server = new FakeServer();
		const stores = memoryStores();
		const inner = stores('alice');
		let calls = 0;
		let late: Promise<unknown> = Promise.resolve();
		let id = '';
		const wrapped = {
			...inner,
			listOutbox: async () => {
				const rows = await inner.listOutbox();
				if (++calls === 2) {
					late = (kind === 'update' ? a.update('todos', id, { title: 'late' }) : a.remove('todos', id)).catch((e: Error) => e);
				}
				return rows;
			}
		};
		const a = device(server, (o) => (o === 'alice' ? wrapped : stores(o)));
		await a.attach('alice');
		id = (await a.create('todos', { title: 'x' })).id;
		await a.flush();
		calls = 0;
		const result = await a.signOut();
		const outcome = await late;
		expect(result.ok === false || (outcome instanceof Error && /signing out/.test(outcome.message))).toBe(true);
	});
});

describe('round 2: failed entries', () => {
	it('retryFailed starts the attempts again, so one 503 does not fail it straight back', async () => {
		let clock = 0;
		const server = new FakeServer();
		const a = createSyncEngine({ stores: memoryStores(), transport: server, collections: ['todos'], lock: (_n, fn) => fn(), now: () => clock });
		await a.attach('alice');
		await a.create('todos', { title: 'p' });
		server.failWhen = () => new ApiError(500, null);
		for (let i = 0; i < 8; i++) {
			clock += 10 * 60_000;
			await a.sync();
		}
		expect((await a.status()).failed).toBe(1);
		server.failWhen = null;
		await a.retryFailed();
		server.failNext = new ApiError(503, null);
		await a.sync();
		const s = await a.status();
		expect(s.failed).toBe(0);
		expect(s.pending).toBe(1);
		clock += 10 * 60_000;
		await a.sync();
		expect(server.records('alice', 'todos')).toHaveLength(1);
	});

	it('backs off for seconds, not milliseconds', async () => {
		let clock = 1_000_000;
		const server = new FakeServer();
		const a = createSyncEngine({ stores: memoryStores(), transport: server, collections: ['todos'], lock: (_n, fn) => fn(), now: () => clock });
		await a.attach('alice');
		await a.create('todos', { title: 'p' });
		server.failNext = new ApiError(500, null);
		await a.sync();
		clock += 1000;
		await a.sync();
		expect(server.pushes).toHaveLength(0); // still backing off after 1s
		clock += 3000;
		await a.sync();
		expect(server.pushes).toHaveLength(1);
	});

	it('is not stuck if the clock jumps back', async () => {
		let clock = 10 * 86_400_000;
		const server = new FakeServer();
		const a = createSyncEngine({ stores: memoryStores(), transport: server, collections: ['todos'], lock: (_n, fn) => fn(), now: () => clock });
		await a.attach('alice');
		await a.create('todos', { title: 'p' });
		server.failNext = new ApiError(500, null);
		await a.sync();
		clock -= 86_400_000;
		await a.sync();
		expect(server.pushes).toHaveLength(1);
	});

	it('discarding a failed create also drops the updates queued behind it', async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		const todo = await a.create('todos', { title: 'x' });
		server.failNext = new ApiError(400, null);
		await a.flush();
		await a.update('todos', todo.id, { title: 'y' });
		const [bad] = await a.failures();
		await a.discardFailed(bad.id);
		await a.flush();
		expect(server.pushes).toEqual([]);
		expect(await a.status()).toMatchObject({ pending: 0, failed: 0 });
	});
});

describe('round 2: pull holds the cursor for a skipped upsert', () => {
	it("delivers another device's fields once the queued change is discarded", async () => {
		const server = new FakeServer();
		const a = device(server);
		await a.attach('alice');
		server.seed('alice', 'todos', { id: 'r1', title: 'one' });
		await a.sync();
		await a.update('todos', 'r1', { title: 'mine' });
		server.failNext = new ApiError(422, null);
		await a.sync();
		server.seed('alice', 'todos', { id: 'r1', title: 'one', extra: 'from the other device' });
		await a.sync();
		const [bad] = await a.failures();
		await a.discardFailed(bad.id);
		await a.sync();
		expect((await a.get('todos', 'r1'))?.extra).toBe('from the other device');
	});
});
