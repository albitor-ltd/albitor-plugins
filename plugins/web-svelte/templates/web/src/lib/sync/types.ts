// Managed by Albitor (web-svelte, offline data layer): the shapes @albitor/sync passes around.

/** A synced record. `id` is chosen on the device, so a create can be replayed safely. */
export interface SyncRecord {
	id: string;
	[field: string]: unknown;
}

export type OutboxOp = 'create' | 'update' | 'delete';

/**
 * One queued change. `rev` counts amendments: an entry that is still `pending` absorbs later edits
 * to the same record and bumps `rev`; an `inflight` entry is never amended, so no edit can be
 * dropped when the entry completes.
 */
export interface OutboxEntry {
	id: number;
	collection: string;
	entityId: string;
	op: OutboxOp;
	/** The whole record for a create, the changed fields for an update, nothing for a delete. */
	payload?: Record<string, unknown>;
	rev: number;
	state: 'pending' | 'inflight' | 'failed';
	attempts: number;
	/** Do not send before this time (ms): the backoff after a server error. */
	retryAt?: number;
	/** Plain words for a `failed` entry; never a response body. */
	error?: string;
	createdAt: number;
}

/**
 * Everything one signed-in person has on this device. A store belongs to exactly one owner, so
 * another person's data can never be read through it.
 */
export interface LocalStore {
	getRecord(collection: string, id: string): Promise<SyncRecord | undefined>;
	putRecord(collection: string, record: SyncRecord): Promise<void>;
	deleteRecord(collection: string, id: string): Promise<void>;
	listRecords(collection: string): Promise<SyncRecord[]>;
	/** Oldest first. */
	listOutbox(): Promise<OutboxEntry[]>;
	/** Stores the entry; one without an id is given the next one. Returns the id. */
	putOutbox(entry: Omit<OutboxEntry, 'id'> & { id?: number }): Promise<number>;
	deleteOutbox(id: number): Promise<void>;
	getCursor(collection: string): Promise<string | null>;
	setCursor(collection: string, cursor: string | null): Promise<void>;
	/** Removes everything, queued work included. Only an explicit discard calls this. */
	clear(): Promise<void>;
}

/** Opens (or creates) the store for one owner. */
export type StoreFactory = (owner: string) => LocalStore;

export interface PullResult {
	upserts: SyncRecord[];
	/** Ids deleted on the server since the cursor: the tombstones other devices must apply. */
	deleted: string[];
	cursor: string | null;
}

/**
 * How changes reach the server. `owner` is who the engine believes is signed in; a transport
 * whose credentials belong to someone else must refuse. Throws the shell's `ApiError` or
 * `NetworkError`.
 */
export interface Transport {
	push(entry: OutboxEntry, owner: string): Promise<void>;
	pull(collection: string, cursor: string | null, owner: string): Promise<PullResult>;
}

export interface SyncStatus {
	owner: string | null;
	/** Changes waiting to upload or in flight (failed ones are counted in `failed`). */
	pending: number;
	failed: number;
	syncing: boolean;
	/** The last run stopped because the server wanted a fresh sign-in. */
	needsSignIn: boolean;
}

export type FlushStop = 'offline' | 'auth' | 'retry' | 'detached';

export interface RunResult {
	pushed: number;
	failed: number;
	stoppedBy?: FlushStop;
}

export type SignOutResult = { ok: true } | { ok: false; pending: number };
