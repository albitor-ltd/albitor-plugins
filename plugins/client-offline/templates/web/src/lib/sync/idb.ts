// Managed by Albitor (client-offline): the IndexedDB LocalStore. One database per owner, so
// another person's data is not merely hidden but unreachable through the store being used.
import type { LocalStore, OutboxEntry, StoreFactory, SyncRecord } from './types';

const VERSION = 1;
const RECORDS = 'records';
const OUTBOX = 'outbox';
const CURSORS = 'cursors';

/** The database name for an owner. The id is encoded so no id can collide with another's name. */
export const dbName = (prefix: string, owner: string) => `${prefix}:${encodeURIComponent(owner)}`;

function wrap<T>(request: IDBRequest<T>): Promise<T> {
	return new Promise((resolve, reject) => {
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
	});
}

function done(tx: IDBTransaction): Promise<void> {
	return new Promise((resolve, reject) => {
		tx.oncomplete = () => resolve();
		tx.onerror = () => reject(tx.error ?? new Error('IndexedDB transaction failed'));
		tx.onabort = () => reject(tx.error ?? new Error('IndexedDB transaction aborted'));
	});
}

function open(factory: IDBFactory, name: string): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const req = factory.open(name, VERSION);
		req.onupgradeneeded = () => {
			const db = req.result;
			db.createObjectStore(RECORDS, { keyPath: ['collection', 'id'] }).createIndex('collection', 'collection');
			db.createObjectStore(OUTBOX, { keyPath: 'id', autoIncrement: true });
			db.createObjectStore(CURSORS);
		};
		req.onsuccess = () => resolve(req.result);
		req.onerror = () => reject(req.error ?? new Error('could not open the local database'));
		req.onblocked = () => reject(new Error('the local database is blocked by another tab'));
	});
}

export function idbStore(name: string, factory: IDBFactory = indexedDB): LocalStore {
	let opened: Promise<IDBDatabase> | null = null;
	const db = () => (opened ??= open(factory, name));
	async function tx<T>(stores: string | string[], mode: IDBTransactionMode, fn: (t: IDBTransaction) => Promise<T> | T): Promise<T> {
		const t = (await db()).transaction(stores, mode);
		const finished = done(t);
		const value = await fn(t);
		await finished;
		return value;
	}
	return {
		getRecord: (c, id) =>
			tx(RECORDS, 'readonly', async (t) => {
				const row = (await wrap(t.objectStore(RECORDS).get([c, id]))) as { record: SyncRecord } | undefined;
				return row?.record;
			}),
		putRecord: (c, record) =>
			tx(RECORDS, 'readwrite', async (t) => {
				await wrap(t.objectStore(RECORDS).put({ collection: c, id: record.id, record }));
			}),
		deleteRecord: (c, id) =>
			tx(RECORDS, 'readwrite', async (t) => {
				await wrap(t.objectStore(RECORDS).delete([c, id]));
			}),
		listRecords: (c) =>
			tx(RECORDS, 'readonly', async (t) => {
				const rows = (await wrap(t.objectStore(RECORDS).index('collection').getAll(c))) as Array<{ record: SyncRecord }>;
				return rows.map((r) => r.record);
			}),
		listOutbox: () =>
			tx(OUTBOX, 'readonly', async (t) => (await wrap(t.objectStore(OUTBOX).getAll())) as OutboxEntry[]),
		putOutbox: (entry) =>
			tx(OUTBOX, 'readwrite', async (t) => (await wrap(t.objectStore(OUTBOX).put(entry))) as number),
		deleteOutbox: (id) =>
			tx(OUTBOX, 'readwrite', async (t) => {
				await wrap(t.objectStore(OUTBOX).delete(id));
			}),
		getCursor: (c) =>
			tx(CURSORS, 'readonly', async (t) => ((await wrap(t.objectStore(CURSORS).get(c))) as string | undefined) ?? null),
		setCursor: (c, cursor) =>
			tx(CURSORS, 'readwrite', async (t) => {
				await wrap(t.objectStore(CURSORS).put(cursor, c));
			}),
		clear: () =>
			tx([RECORDS, OUTBOX, CURSORS], 'readwrite', async (t) => {
				await Promise.all([RECORDS, OUTBOX, CURSORS].map((s) => wrap(t.objectStore(s).clear())));
			})
	};
}

/** One database per owner, named `<prefix>:<owner>`. */
export function idbStores(prefix = 'albitor-sync', factory?: IDBFactory): StoreFactory {
	const byOwner = new Map<string, LocalStore>();
	return (owner) => {
		let store = byOwner.get(owner);
		if (!store) byOwner.set(owner, (store = idbStore(dbName(prefix, owner), factory)));
		return store;
	};
}
