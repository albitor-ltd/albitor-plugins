// Starting point (test-playwright).
import { randomUUID } from "node:crypto";
import type { APIRequestContext } from "@playwright/test";

// The hosted function runs at most 10 calls at once; over it the API answers 503 (a throttled call
// never ran the function, so the POST is safe to repeat). 429 is the API's rate limit, same rule.
const RETRY_STATUSES = new Set([429, 503]);
const DEFAULT_RETRY = { retries: 3, baseDelayMs: 250, maxDelayMs: 5_000 };
const DEFAULT_CONCURRENCY = 4;

export type FixtureScopeOptions = {
	retries?: number;
	baseDelayMs?: number;
	maxDelayMs?: number;
	// Replaceable so the harness's own tests run without waiting.
	sleep?: (ms: number) => Promise<void>;
};

// One test owns one scope. Names differ across workers, retries and separate runs; cleanup
// deletes only exact records created through the authenticated API. Never adopt ids from a list
// response.
export class FixtureScope {
	readonly namespace = randomUUID();
	private readonly owned = new Set<string>();
	private sequence = 0;
	// Records the server may hold but this scope cannot name, so cannot delete.
	private readonly leaked: string[] = [];

	private readonly request: APIRequestContext;
	private readonly headers: Record<string, string>;

	private readonly retry: Required<Omit<FixtureScopeOptions, "sleep">>;
	private readonly sleep: (ms: number) => Promise<void>;

	constructor(request: APIRequestContext, headers: Record<string, string> = {}, options: FixtureScopeOptions = {}) {
		this.request = request;
		this.headers = headers;
		this.retry = {
			retries: options.retries ?? DEFAULT_RETRY.retries,
			baseDelayMs: options.baseDelayMs ?? DEFAULT_RETRY.baseDelayMs,
			maxDelayMs: options.maxDelayMs ?? DEFAULT_RETRY.maxDelayMs,
		};
		this.sleep = options.sleep ?? ((ms) => new Promise((done) => setTimeout(done, ms)));
	}

	// Full-jitter exponential backoff, bounded; a Retry-After in seconds sets the floor.
	private delayFor(attempt: number, retryAfter: string | undefined): number {
		const { baseDelayMs, maxDelayMs } = this.retry;
		const jittered = Math.random() * Math.min(maxDelayMs, baseDelayMs * 2 ** attempt);
		const seconds = retryAfter === undefined ? NaN : Number(retryAfter);
		const hinted = Number.isFinite(seconds) && seconds >= 0 ? seconds * 1000 : 0;
		return Math.min(maxDelayMs, Math.max(jittered, hinted));
	}

	uniqueName(prefix = "fixture"): string {
		this.sequence += 1;
		const suffix = `-${this.namespace}-${this.sequence}`;
		const safePrefix = (prefix || "fixture").slice(0, Math.max(1, 80 - suffix.length));
		return `${safePrefix}${suffix}`;
	}

	async create<T extends { id: string }>(collection: string, data: unknown): Promise<T> {
		if (!/^\/api\/[a-zA-Z0-9/_-]+$/.test(collection) || collection.includes("..")) {
			throw new Error(`Fixture collection must be a relative API path: ${collection}`);
		}
		let response = await this.request.post(collection, { data, headers: this.headers });
		for (let attempt = 0; attempt < this.retry.retries && RETRY_STATUSES.has(response.status()); attempt++) {
			await this.sleep(this.delayFor(attempt, response.headers()["retry-after"]));
			response = await this.request.post(collection, { data, headers: this.headers });
		}
		if (!response.ok()) {
			throw new Error(`Fixture creation failed: ${response.status()} ${await response.text()}`);
		}
		let record: T;
		try {
			record = (await response.json()) as T;
		} catch {
			this.leaked.push(`${collection}: a 2xx answer that is not JSON`);
			throw new Error(`Fixture creation answered ${response.status()} but the record cannot be cleaned up: ${collection}`);
		}
		if (typeof record?.id !== "string" || !record.id || record.id === "." || record.id === "..") {
			this.leaked.push(`${collection}: a 2xx answer with no usable id`);
			throw new Error(`Fixture creation returned no usable id: ${collection}`);
		}
		this.owned.add(`${collection.replace(/\/$/, "")}/${encodeURIComponent(record.id)}`);
		return record;
	}

	// Bulk seeding: never more than `concurrency` creates in flight (default 4, well under the hosted
	// function's 10), results in input order. After a failure no new create starts, in-flight ones
	// finish so everything created is owned and cleaned up, and the first error is thrown.
	async createMany<T extends { id: string }>(
		collection: string,
		items: readonly unknown[],
		options: { concurrency?: number } = {},
	): Promise<T[]> {
		const concurrency = Math.max(1, Math.floor(options.concurrency ?? DEFAULT_CONCURRENCY));
		const results: T[] = new Array(items.length);
		let next = 0;
		let failure: { error: unknown } | undefined;
		const lane = async () => {
			while (!failure && next < items.length) {
				const index = next++;
				try {
					results[index] = await this.create<T>(collection, items[index]);
				} catch (error) {
					failure ??= { error };
				}
			}
		};
		await Promise.all(Array.from({ length: Math.min(concurrency, items.length) }, lane));
		if (failure) throw failure.error;
		return results;
	}

	async remove(path: string): Promise<void> {
		if (!this.owned.has(path)) {
			throw new Error(`Refusing to delete an unowned fixture: ${path}`);
		}
		const response = await this.request.delete(path, { headers: this.headers });
		if (response.status() === 404) {
			// Some APIs answer 404 for an unsupported DELETE too. Confirm absence instead of
			// reporting successful cleanup while the record remains.
			const remaining = await this.request.get(path, { headers: this.headers });
			if (remaining.status() !== 404) {
				throw new Error(`Fixture was not deleted: ${path}`);
			}
		} else if (!response.ok()) {
			throw new Error(`Fixture cleanup failed: ${response.status()} ${path}`);
		}
		this.owned.delete(path);
	}

	async cleanup(): Promise<void> {
		const failures: unknown[] = [];
		// Remove children before parents. Try all removals and report leaks.
		for (const path of [...this.owned].reverse()) {
			try {
				await this.remove(path);
			} catch (error) {
				failures.push(error);
			}
		}
		for (const what of this.leaked) {
			failures.push(new Error(`Unnamed fixture left behind: ${what}`));
		}
		if (failures.length) {
			throw new AggregateError(failures, "Fixture cleanup left owned records behind");
		}
	}
}
