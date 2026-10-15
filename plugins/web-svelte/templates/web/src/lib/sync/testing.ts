// Managed by Albitor (web-svelte, offline data layer): an in-memory server for tests, speaking the Transport
// contract. It keeps one account per owner, with tombstones for deletions.
import { ApiError, NetworkError } from '../shell/api';
import type { OutboxEntry, PullResult, SyncRecord, Transport } from './types';

interface Row {
	record: SyncRecord;
	seq: number;
}

export class FakeServer implements Transport {
	private rows = new Map<string, Map<string, Row>>();
	private tombstones = new Map<string, Array<{ seq: number; id: string }>>();
	private seq = 0;
	/** Every push that reached the server, in order. */
	pushes: Array<{ owner: string; entry: OutboxEntry }> = [];
	offline = false;
	/** Set to hold every push until it resolves. */
	gate: Promise<void> | null = null;
	/** Throws this from the next push, once. */
	failNext: Error | null = null;
	/** Throws what this returns for any push it names, every time. */
	failWhen: ((entry: OutboxEntry) => Error | null) | null = null;
	active = 0;
	maxActive = 0;

	private table(owner: string, collection: string): Map<string, Row> {
		const key = `${owner}\u0000${collection}`;
		let t = this.rows.get(key);
		if (!t) this.rows.set(key, (t = new Map()));
		return t;
	}

	/** Puts a record on the server as if another device had written it. */
	seed(owner: string, collection: string, record: SyncRecord): void {
		this.table(owner, collection).set(record.id, { record: { ...record }, seq: ++this.seq });
	}

	/** Deletes on the server as if another device had, leaving a tombstone. */
	removeRemote(owner: string, collection: string, id: string): void {
		this.table(owner, collection).delete(id);
		const key = `${owner}\u0000${collection}`;
		this.tombstones.set(key, [...(this.tombstones.get(key) ?? []), { seq: ++this.seq, id }]);
	}

	records(owner: string, collection: string): SyncRecord[] {
		return [...this.table(owner, collection).values()].map((r) => ({ ...r.record }));
	}

	async push(entry: OutboxEntry, owner: string): Promise<void> {
		this.active++;
		this.maxActive = Math.max(this.maxActive, this.active);
		try {
			if (this.gate) await this.gate;
			if (this.offline) throw new NetworkError(true);
			const always = this.failWhen?.(entry);
			if (always) throw always;
			if (this.failNext) {
				const err = this.failNext;
				this.failNext = null;
				throw err;
			}
			this.pushes.push({ owner, entry: structuredClone(entry) });
			const rows = this.table(owner, entry.collection);
			const row = rows.get(entry.entityId);
			if (entry.op === 'create') {
				if (row) throw new ApiError(409, null);
				rows.set(entry.entityId, { record: { ...entry.payload, id: entry.entityId }, seq: ++this.seq });
			} else if (entry.op === 'update') {
				if (!row) throw new ApiError(404, null);
				rows.set(entry.entityId, { record: { ...row.record, ...entry.payload }, seq: ++this.seq });
			} else {
				if (!row) throw new ApiError(404, null);
				this.removeRemote(owner, entry.collection, entry.entityId);
			}
		} finally {
			this.active--;
		}
	}

	async pull(collection: string, cursor: string | null, owner: string): Promise<PullResult> {
		if (this.offline) throw new NetworkError(true);
		const since = cursor === null ? 0 : Number(cursor);
		const live = [...this.table(owner, collection).values()].filter((r) => r.seq > since);
		const dead = (this.tombstones.get(`${owner}\u0000${collection}`) ?? []).filter((t) => t.seq > since);
		return {
			upserts: live.map((r) => ({ ...r.record })),
			deleted: dead.map((t) => t.id),
			cursor: String(Math.max(since, ...live.map((r) => r.seq), ...dead.map((t) => t.seq)))
		};
	}
}
