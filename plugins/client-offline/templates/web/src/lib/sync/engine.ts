// Managed by Albitor (client-offline): the sync engine, @albitor/sync.
//
// Offline-first: every write lands in the local store first and in an outbox second, and the
// outbox is replayed to the server when there is a network. The rules below exist because
// regenerated sync layers got each of them wrong; each has a test in engine.test.ts.
//
//  - A queued entry that is still pending absorbs later edits; one that is in flight is never
//    touched, and an edit made meanwhile becomes its own entry. So no edit is lost when the
//    in-flight entry completes.
//  - Local data is partitioned by signed-in person. The engine only ever reads, pushes or pulls
//    through the store of the person attached now, so a different person on the same device sees
//    nothing of the last person's, and the last person's queue never uploads into the new account.
//  - A deletion is an outbox entry like any other and reaches other devices as a tombstone in pull.
//  - Signing out never discards unsynced work unless the caller says so.
//  - Every upload and download runs inside one guarded run per person (and one cross-tab lock).
import { ApiError, NetworkError } from '../shell/api';
import type {
	FlushStop,
	LocalStore,
	OutboxEntry,
	RunResult,
	SignOutResult,
	StoreFactory,
	SyncRecord,
	SyncStatus,
	Transport
} from './types';

export type ChangeKind = 'write' | 'status';

export type Lock = <T>(name: string, fn: () => Promise<T>) => Promise<T>;

export interface SyncEngineOptions {
	stores: StoreFactory;
	transport: Transport;
	/** The collections this app syncs. Anything else is refused. */
	collections: readonly string[];
	/** Serialises runs across tabs. Defaults to the Web Locks API when the browser has it. */
	lock?: Lock;
	now?: () => number;
	newId?: () => string;
}

export interface FailedChange {
	id: number;
	collection: string;
	entityId: string;
	op: OutboxEntry['op'];
	error: string;
}

interface Ctx {
	owner: string;
	store: LocalStore;
	/** Serialises reads and writes of the outbox, so no edit can interleave with a state change. */
	chain: Promise<unknown>;
	running: Promise<RunResult> | null;
	again: boolean;
	wantPull: boolean;
	needsSignIn: boolean;
	/** Set once sign-out has decided; later writes are refused, not written to a store being wiped. */
	closed: boolean;
}

/** Server errors are retried this many times in all before the change is marked failed. */
export const MAX_ATTEMPTS = 5;
const MAX_BACKOFF = 5 * 60_000;
const backoff = (attempts: number) => Math.min(2000 * 2 ** (attempts - 1), MAX_BACKOFF);

const webLock: Lock = (name, fn) =>
	typeof navigator !== 'undefined' && navigator.locks ? navigator.locks.request(name, fn) : fn();

type Outcome = 'done' | 'offline' | 'auth' | 'retry' | 'fail';

/** What a failed upload means. Only a change the server will never accept is `fail`. */
function classify(err: unknown, op: OutboxEntry['op']): Outcome {
	// Known gap: a request that times out before any response arrives also lands here, so a server
	// that hangs for ever pauses the whole queue without counting attempts.
	if (err instanceof NetworkError) return 'offline';
	if (err instanceof ApiError) {
		const s = err.status;
		if (s === 401) return 'auth';
		// A create replayed after an unseen success, or a delete of what is already gone.
		if ((s === 409 && op === 'create') || (s === 404 && op === 'delete')) return 'done';
		if (s >= 500 || s === 408 || s === 429) return 'retry';
		return 'fail';
	}
	// A TypeError is the API client refusing the request itself (a bad path): it would fail identically.
	return err instanceof TypeError ? 'fail' : 'retry';
}

const stopFor = (o: Outcome): FlushStop => (o === 'offline' ? 'offline' : o === 'auth' ? 'auth' : 'retry');

/** Ids go into URLs and database keys, so they stay to a plain alphabet. */
export const isRecordId = (id: unknown): id is string => typeof id === 'string' && ID.test(id);
const ID = /^[A-Za-z0-9_-]{1,64}$/;

const keyOf = (e: Pick<OutboxEntry, 'collection' | 'entityId'>) => `${e.collection}\u0000${e.entityId}`;

export interface SyncEngine {
	readonly owner: string | null;
	/** Start working as `owner`. Another person's data on this device stays where it is. */
	attach(owner: string): Promise<void>;
	/** Stop working as the current person without touching their data or queue. */
	detach(): void;
	list(collection: string): Promise<SyncRecord[]>;
	get(collection: string, id: string): Promise<SyncRecord | undefined>;
	create(collection: string, fields: Record<string, unknown>): Promise<SyncRecord>;
	update(collection: string, id: string, patch: Record<string, unknown>): Promise<SyncRecord>;
	remove(collection: string, id: string): Promise<void>;
	/** Upload the outbox. Concurrent callers share one run. */
	flush(): Promise<RunResult>;
	/** Upload, then download what changed. Concurrent callers share one run. */
	sync(): Promise<RunResult>;
	status(): Promise<SyncStatus>;
	failures(): Promise<FailedChange[]>;
	retryFailed(): Promise<void>;
	/** Give up on one rejected change (and the record it created, if it was a create). */
	discardFailed(id: number): Promise<void>;
	/**
	 * Upload what can be uploaded, then sign out only if nothing is left. Otherwise nothing changes
	 * and the count is returned for the person to decide. `discard` throws the queue away.
	 */
	signOut(options?: { discard?: boolean }): Promise<SignOutResult>;
	/** `write` means the local data or queue changed; `status` means only the sync state did. */
	subscribe(listener: (kind: ChangeKind) => void): () => void;
}

export function createSyncEngine(options: SyncEngineOptions): SyncEngine {
	const { stores, transport, collections } = options;
	const lock = options.lock ?? webLock;
	const now = options.now ?? Date.now;
	const newId = options.newId ?? (() => crypto.randomUUID());
	const listeners = new Set<(kind: ChangeKind) => void>();
	let attachGen = 0;
	let current: Ctx | null = null;

	const emit = (kind: ChangeKind = 'status') => listeners.forEach((l) => l(kind));

	function need(collection?: string): Ctx {
		if (!current) throw new Error('not signed in: attach a person before reading or writing');
		if (collection !== undefined && !collections.includes(collection)) {
			throw new Error(`unknown collection ${JSON.stringify(collection)}`);
		}
		return current;
	}

	function exclusive<T>(ctx: Ctx, fn: () => Promise<T>): Promise<T> {
		const p = ctx.chain.then(fn);
		ctx.chain = p.catch(() => undefined);
		return p;
	}

	async function pushAll(ctx: Ctx): Promise<RunResult> {
		const result: RunResult = { pushed: 0, failed: 0 };
		let retried = false;
		for (;;) {
			if (current !== ctx) return { ...result, stoppedBy: 'detached' };
			const entry = await exclusive(ctx, async () => {
				const outbox = await ctx.store.listOutbox();
				// A rejected change blocks later changes to the same record, which depend on it.
				// A change backing off after a server error blocks them too, until its time comes. A
				// retryAt further off than the longest backoff means the clock went back: it is due.
				const t = now();
				const blocked = new Set(
					outbox.filter((e) => e.state === 'failed' || (e.state === 'pending' && (e.retryAt ?? 0) > t && (e.retryAt ?? 0) - t <= MAX_BACKOFF)).map(keyOf)
				);
				const next = outbox.find((e) => e.state === 'pending' && !blocked.has(keyOf(e)));
				if (!next || ctx.closed) return undefined;
				const sent = { ...next, state: 'inflight' as const };
				await ctx.store.putOutbox(sent);
				return sent;
			});
			if (!entry) return retried ? { ...result, stoppedBy: 'retry' } : result;

			let outcome: Outcome = 'done';
			try {
				await transport.push(entry, ctx.owner);
			} catch (err) {
				outcome = classify(err, entry.op);
				if (outcome === 'fail') {
					await exclusive(ctx, async () => {
						// Sign-out has wiped this store: writing now would resurrect discarded work.
						if (ctx.closed) return;
						await ctx.store.putOutbox({
							...entry,
							state: 'failed',
							attempts: entry.attempts + 1,
							error: `The server rejected this change${err instanceof ApiError ? ` (${err.status})` : ''}.`
						});
					});
					result.failed++;
					continue;
				}
			}

			await exclusive(ctx, async () => {
				const live = (await ctx.store.listOutbox()).find((e) => e.id === entry.id);
				if (!live || ctx.closed) return;
				if (outcome === 'done' && live.rev === entry.rev) {
					await ctx.store.deleteOutbox(entry.id);
				} else if (outcome === 'retry') {
					const attempts = live.attempts + 1;
					await ctx.store.putOutbox(
						attempts >= MAX_ATTEMPTS
							? { ...live, state: 'failed', attempts, error: 'The server kept failing to take this change.' }
							: { ...live, state: 'pending', attempts, retryAt: now() + backoff(attempts) }
					);
				} else {
					// Not delivered, or amended since it was sent: it goes back to be sent again.
					await ctx.store.putOutbox({ ...live, state: 'pending' });
				}
			});
			if (outcome === 'done') {
				result.pushed++;
				emit();
				continue;
			}
			if (outcome === 'retry') {
				// Only this record is held back; the rest of the queue goes on.
				retried = true;
				emit();
				continue;
			}
			return { ...result, stoppedBy: stopFor(outcome) };
		}
	}

	async function pullAll(ctx: Ctx): Promise<FlushStop | undefined> {
		for (const collection of collections) {
			const cursor = await ctx.store.getCursor(collection);
			let page;
			try {
				page = await transport.pull(collection, cursor, ctx.owner);
			} catch (err) {
				return stopFor(classify(err, 'update'));
			}
			if (current !== ctx) return 'detached';
			await exclusive(ctx, async () => {
				// A record with queued local changes keeps them, and the cursor stays put so the server's
				// copy (or tombstone) is fetched again until the queue for that record is resolved.
				const touched = new Set((await ctx.store.listOutbox()).filter((e) => e.collection === collection).map((e) => e.entityId));
				let held = false;
				for (const record of page.upserts) {
					if (touched.has(record.id)) held = true;
					else await ctx.store.putRecord(collection, record);
				}
				for (const id of page.deleted) {
					if (touched.has(id)) held = true;
					else await ctx.store.deleteRecord(collection, id);
				}
				// A permanently failed entry therefore keeps this collection's cursor back, and each pull
				// re-downloads the delta, until the person retries or discards it.
				if (!held) await ctx.store.setCursor(collection, page.cursor);
			});
		}
		return undefined;
	}

	function run(ctx: Ctx, pull: boolean): Promise<RunResult> {
		if (ctx.running) {
			// Someone is already uploading: fold this request into that run.
			ctx.again = true;
			ctx.wantPull ||= pull;
			return ctx.running;
		}
		ctx.wantPull = pull;
		const body = async (): Promise<RunResult> => {
			const total: RunResult = { pushed: 0, failed: 0 };
			ctx.needsSignIn = false;
			emit();
			do {
				ctx.again = false;
				const doPull = ctx.wantPull;
				ctx.wantPull = false;
				const r = await pushAll(ctx);
				total.pushed += r.pushed;
				total.failed += r.failed;
				if (r.stoppedBy) {
					total.stoppedBy = r.stoppedBy;
					break;
				}
				if (doPull) {
					const stop = await pullAll(ctx);
					if (stop) {
						total.stoppedBy = stop;
						break;
					}
				}
			} while (ctx.again);
			if (total.stoppedBy === 'auth') ctx.needsSignIn = true;
			return total;
		};
		const running = Promise.resolve()
			.then(() => lock(`albitor-sync:${ctx.owner}`, body))
			.finally(() => {
				ctx.running = null;
				emit();
			});
		ctx.running = running;
		return running;
	}

	const engine: SyncEngine = {
		get owner() {
			return current?.owner ?? null;
		},

		async attach(owner) {
			if (typeof owner !== 'string' || owner === '') throw new Error('attach needs a person id');
			if (current?.owner === owner) return;
			// A later attach or detach supersedes this one, however long the store takes to open.
			const gen = ++attachGen;
			current = null;
			const store = stores(owner);
			const ctx: Ctx = { owner, store, chain: Promise.resolve(), running: null, again: false, wantPull: false, needsSignIn: false, closed: false };
			// Entries left in flight by a closed tab or a crash are sent again; creates are idempotent.
			for (const e of await store.listOutbox()) {
				if (e.state === 'inflight') await store.putOutbox({ ...e, state: 'pending' });
			}
			if (gen !== attachGen) return;
			current = ctx;
			emit('write');
		},

		detach() {
			attachGen++;
			current = null;
			emit('write');
		},

		async list(collection) {
			const ctx = need(collection);
			return ctx.store.listRecords(collection);
		},

		async get(collection, id) {
			const ctx = need(collection);
			return ctx.store.getRecord(collection, id);
		},

		async create(collection, fields) {
			const ctx = need(collection);
			const { id: given, ...rest } = fields;
			const id = typeof given === 'string' && given !== '' ? given : newId();
			if (!isRecordId(id)) throw new Error('a record id may use letters, digits, - and _ (up to 64)');
			const record: SyncRecord = { ...rest, id };
			await exclusive(ctx, async () => {
				if (ctx.closed) throw new Error('signing out: this write was not saved');
				if (await ctx.store.getRecord(collection, id)) throw new Error(`${collection}/${id} already exists`);
				await ctx.store.putRecord(collection, record);
				await ctx.store.putOutbox({ collection, entityId: id, op: 'create', payload: rest, rev: 1, state: 'pending', attempts: 0, createdAt: now() });
			});
			emit('write');
			return record;
		},

		async update(collection, id, patch) {
			const ctx = need(collection);
			const { id: _ignored, ...changes } = patch;
			void _ignored;
			const record = await exclusive(ctx, async () => {
				if (ctx.closed) throw new Error('signing out: this write was not saved');
				const existing = await ctx.store.getRecord(collection, id);
				if (!existing) throw new Error(`no such record ${collection}/${id}`);
				const next: SyncRecord = { ...existing, ...changes, id };
				await ctx.store.putRecord(collection, next);
				const mine = (await ctx.store.listOutbox()).filter((e) => e.collection === collection && e.entityId === id);
				const last = mine[mine.length - 1];
				if (last && last.state === 'pending' && last.op !== 'delete') {
					// Not sent yet: fold the edit into it. An in-flight entry is never amended.
					await ctx.store.putOutbox({ ...last, payload: { ...last.payload, ...changes }, rev: last.rev + 1 });
				} else {
					await ctx.store.putOutbox({ collection, entityId: id, op: 'update', payload: changes, rev: 1, state: 'pending', attempts: 0, createdAt: now() });
				}
				return next;
			});
			emit('write');
			return record;
		},

		async remove(collection, id) {
			const ctx = need(collection);
			if (!isRecordId(id)) throw new Error('a record id may use letters, digits, - and _ (up to 64)');
			await exclusive(ctx, async () => {
				if (ctx.closed) throw new Error('signing out: this write was not saved');
				const mine = (await ctx.store.listOutbox()).filter((e) => e.collection === collection && e.entityId === id);
				// If the server never accepted a create, it has nothing to delete.
				const neverSent = mine.some((e) => e.op === 'create' && e.state !== 'inflight');
				for (const e of mine) {
					if (e.state !== 'inflight') await ctx.store.deleteOutbox(e.id);
				}
				await ctx.store.deleteRecord(collection, id);
				if (!neverSent) {
					await ctx.store.putOutbox({ collection, entityId: id, op: 'delete', rev: 1, state: 'pending', attempts: 0, createdAt: now() });
				}
			});
			emit('write');
		},

		async flush() {
			return run(need(), false);
		},
		async sync() {
			return run(need(), true);
		},

		async status() {
			const ctx = current;
			if (!ctx) return { owner: null, pending: 0, failed: 0, syncing: false, needsSignIn: false };
			const outbox = await ctx.store.listOutbox();
			return {
				owner: ctx.owner,
				pending: outbox.filter((e) => e.state !== 'failed').length,
				failed: outbox.filter((e) => e.state === 'failed').length,
				syncing: ctx.running !== null,
				needsSignIn: ctx.needsSignIn
			};
		},

		async failures() {
			const ctx = need();
			return (await ctx.store.listOutbox())
				.filter((e) => e.state === 'failed')
				.map((e) => ({ id: e.id, collection: e.collection, entityId: e.entityId, op: e.op, error: e.error ?? 'The server rejected this change.' }));
		},

		async retryFailed() {
			const ctx = need();
			await exclusive(ctx, async () => {
				for (const e of await ctx.store.listOutbox()) {
					if (e.state === 'failed') {
						await ctx.store.putOutbox({ ...e, state: 'pending', error: undefined, attempts: 0, retryAt: undefined });
					}
				}
			});
			emit('write');
		},

		async discardFailed(id) {
			const ctx = need();
			await exclusive(ctx, async () => {
				const e = (await ctx.store.listOutbox()).find((x) => x.id === id && x.state === 'failed');
				if (!e) return;
				await ctx.store.deleteOutbox(id);
				if (e.op === 'create') {
					// The server never had the record, so changes queued behind its create cannot apply.
					for (const dep of await ctx.store.listOutbox()) {
						if (dep.collection === e.collection && dep.entityId === e.entityId && dep.state !== 'inflight') {
							await ctx.store.deleteOutbox(dep.id);
						}
					}
					await ctx.store.deleteRecord(e.collection, e.entityId);
				}
			});
			emit('write');
		},

		async signOut(opts) {
			const ctx = current;
			if (!ctx) return { ok: true };
			if (!opts?.discard) await run(ctx, false).catch(() => undefined);
			// Counting and clearing are one step in the write queue, so no write can land between them.
			const result = await exclusive(ctx, async (): Promise<SignOutResult> => {
				const left = (await ctx.store.listOutbox()).length;
				if (left > 0 && !opts?.discard) return { ok: false, pending: left };
				// Nothing is unsynced (or the person chose to discard): wiping loses nothing unasked.
				ctx.closed = true;
				try {
					await ctx.store.clear();
				} catch (err) {
					ctx.closed = false; // the store is still usable: do not leave the session refusing writes
					throw err;
				}
				return { ok: true };
			});
			// Someone else may have attached while this was finishing; only end the session that asked.
			if (result.ok && current === ctx) engine.detach();
			return result;
		},

		subscribe(listener) {
			listeners.add(listener);
			return () => listeners.delete(listener);
		}
	};
	return engine;
}
