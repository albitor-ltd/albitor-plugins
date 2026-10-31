// Starting point (web-svelte, offline data layer): an in-memory LocalStore, for tests and for a browser with no
// IndexedDB (private windows, some embedded views). Nothing survives a reload.
import type { LocalStore, OutboxEntry, StoreFactory, SyncRecord } from './types';

export function memoryStore(): LocalStore {
	const records = new Map<string, Map<string, SyncRecord>>();
	const outbox = new Map<number, OutboxEntry>();
	const cursors = new Map<string, string | null>();
	let nextId = 1;
	const table = (c: string) => records.get(c) ?? records.set(c, new Map()).get(c)!;
	return {
		async getRecord(c, id) {
			const r = table(c).get(id);
			return r && structuredClone(r);
		},
		async putRecord(c, record) {
			table(c).set(record.id, structuredClone(record));
		},
		async deleteRecord(c, id) {
			table(c).delete(id);
		},
		async listRecords(c) {
			return [...table(c).values()].map((r) => structuredClone(r));
		},
		async listOutbox() {
			return [...outbox.values()].sort((a, b) => a.id - b.id).map((e) => structuredClone(e));
		},
		async putOutbox(entry) {
			const id = entry.id ?? nextId++;
			if (id >= nextId) nextId = id + 1;
			outbox.set(id, structuredClone({ ...entry, id }));
			return id;
		},
		async deleteOutbox(id) {
			outbox.delete(id);
		},
		async getCursor(c) {
			return cursors.get(c) ?? null;
		},
		async setCursor(c, cursor) {
			cursors.set(c, cursor);
		},
		async clear() {
			records.clear();
			outbox.clear();
			cursors.clear();
		}
	};
}

/** One store per owner, created on first use and kept for the life of the factory. */
export function memoryStores(): StoreFactory {
	const byOwner = new Map<string, LocalStore>();
	return (owner) => byOwner.get(owner) ?? byOwner.set(owner, memoryStore()).get(owner)!;
}
