// Managed by Albitor (client-offline): the default Transport, over the app's own REST API and the
// shell's API client (path checks, bearer token, error types).
//
// The contract an API must meet for a collection `items` (all under /api):
//   POST   /items            {id, ...fields}   create; the id is the device's. A repeat of an id the
//                                              server already has answers 409 (taken as done).
//   PUT    /items/:id        {...changed}      update (partial).
//   DELETE /items/:id                          delete, and keep a tombstone. 404 is taken as done.
//   GET    /items?since=C    {items, deleted, cursor}  what changed after cursor C (everything, when
//                                              absent): changed records, ids deleted since, and the
//                                              cursor to send next time.
import { ApiError, api } from '../shell/api';
import { isRecordId } from './engine';
import { getSyncUser } from './session';
import type { OutboxEntry, PullResult, SyncRecord, Transport } from './types';

export interface ApiTransportOptions {
	/** The collection's path; defaults to `/<collection>`. */
	path?: (collection: string) => string;
	/** Who the API's credentials belong to. Defaults to the user the auth plugin reported. */
	whoami?: () => string | null;
	/** A request that takes longer is abandoned (and retried later); a hung one would hold the sync lock. */
	timeoutMs?: number;
}

/** Parses a pull response, refusing anything that is not the contract. */
export function parsePull(body: unknown): PullResult {
	const b = body as { items?: unknown; deleted?: unknown; cursor?: unknown } | null;
	const items = b?.items;
	const deleted = b?.deleted ?? [];
	const cursor = b?.cursor ?? null;
	const okRecord = (r: unknown): r is SyncRecord =>
		typeof r === 'object' &&
		r !== null &&
		isRecordId((r as { id?: unknown }).id);
	if (
		!Array.isArray(items) ||
		!items.every(okRecord) ||
		!Array.isArray(deleted) ||
		!deleted.every(isRecordId) ||
		!(cursor === null || typeof cursor === 'string')
	) {
		throw new Error('the sync response did not match the contract');
	}
	return { upserts: items, deleted, cursor };
}

export function apiTransport(options: ApiTransportOptions = {}): Transport {
	const base = options.path ?? ((c: string) => `/${c}`);
	const whoami = options.whoami ?? getSyncUser;
	const signal = () => AbortSignal.timeout(options.timeoutMs ?? 30_000);
	// The credentials belong to whoever is signed in now. If that is not the person the queue
	// belongs to, sending would put one person's changes into another's account.
	const check = (owner: string) => {
		if (whoami() !== owner) throw new ApiError(401, null);
	};
	return {
		async push(entry: OutboxEntry, owner: string) {
			check(owner);
			const root = base(entry.collection);
			const one = `${root}/${encodeURIComponent(entry.entityId)}`;
			if (entry.op === 'create') await api(root, { method: 'POST', body: { ...entry.payload, id: entry.entityId }, signal: signal() });
			else if (entry.op === 'update') await api(one, { method: 'PUT', body: entry.payload ?? {}, signal: signal() });
			else await api(one, { method: 'DELETE', signal: signal() });
		},
		async pull(collection: string, cursor: string | null, owner: string) {
			check(owner);
			return parsePull(
				await api(base(collection), { query: cursor === null ? undefined : { since: cursor }, signal: signal() })
			);
		}
	};
}
