// Managed by Albitor (test-playwright).
import { randomUUID } from "node:crypto";
import type { APIRequestContext } from "@playwright/test";

// One test owns one scope. Names differ across workers, retries and separate runs; cleanup
// deletes only exact records created through the authenticated API. Never adopt ids from a list
// response.
export class FixtureScope {
	readonly namespace = randomUUID();
	private readonly owned = new Set<string>();
	private sequence = 0;

	private readonly request: APIRequestContext;
	private readonly headers: Record<string, string>;

	constructor(request: APIRequestContext, headers: Record<string, string> = {}) {
		this.request = request;
		this.headers = headers;
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
		const response = await this.request.post(collection, { data, headers: this.headers });
		if (!response.ok()) {
			throw new Error(`Fixture creation failed: ${response.status()} ${await response.text()}`);
		}
		const record = (await response.json()) as T;
		if (typeof record.id !== "string" || !record.id || record.id === "." || record.id === "..") {
			throw new Error(`Fixture creation returned no usable id: ${collection}`);
		}
		this.owned.add(`${collection.replace(/\/$/, "")}/${encodeURIComponent(record.id)}`);
		return record;
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
		if (failures.length) {
			throw new AggregateError(failures, "Fixture cleanup left owned records behind");
		}
	}
}
