// Starting point (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { FixtureScope } from "../helpers/fixture-scope.ts";

type Call = { method: string; path: string; headers?: Record<string, string> };

// A fake APIRequestContext: just the three verbs FixtureScope uses.
function fake(opts: { createStatus?: number | ((n: number) => number); retryAfter?: string; hold?: () => Promise<void>; deleteStatus?: (path: string) => number; getStatus?: number } = {}) {
	const calls: Call[] = [];
	let next = 0;
	const res = (status: number, body: unknown = {}, headers: Record<string, string> = {}) => ({
		headers: () => headers,
		ok: () => status >= 200 && status < 300,
		status: () => status,
		text: async () => JSON.stringify(body),
		json: async () => body,
	});
	const request = {
		post: async (path: string, o: { headers?: Record<string, string> }) => {
			calls.push({ method: "POST", path, headers: o.headers });
			next += 1;
			const n = next;
			await opts.hold?.();
			const status = typeof opts.createStatus === "function" ? opts.createStatus(n) : (opts.createStatus ?? 201);
			return res(status, { id: `id${n}` }, opts.retryAfter ? { "retry-after": opts.retryAfter } : {});
		},
		delete: async (path: string, o: { headers?: Record<string, string> }) => {
			calls.push({ method: "DELETE", path, headers: o.headers });
			return res(opts.deleteStatus?.(path) ?? 204);
		},
		get: async (path: string, o: { headers?: Record<string, string> }) => {
			calls.push({ method: "GET", path, headers: o.headers });
			return res(opts.getStatus ?? 404);
		},
	};
	return { request: request as never, calls };
}

describe("FixtureScope", () => {
	it("refuses a collection that is not a relative API path", async () => {
		const { request, calls } = fake();
		const scope = new FixtureScope(request);
		for (const bad of ["/api/../admin", "https://evil.example/api/x", "api/items", "/other/items", "/api/it ems", "/api/"]) {
			await assert.rejects(scope.create(bad, {}), /relative API path/, bad);
		}
		assert.equal(calls.length, 0);
	});

	it("refuses a record whose id is missing or a dot segment", async () => {
		for (const id of ["", ".", ".."]) {
			const { request } = fake();
			(request as { post: unknown }).post = async () => ({
				ok: () => true,
				status: () => 201,
				text: async () => "",
				json: async () => ({ id }),
			});
			await assert.rejects(new FixtureScope(request).create("/api/items", {}), /no usable id/);
		}
	});

	it("deletes only what it created, children before parents", async () => {
		const { request, calls } = fake();
		const scope = new FixtureScope(request, { Authorization: "Bearer t" });
		await scope.create("/api/items", {});
		await scope.create("/api/items", {});
		await scope.cleanup();
		const deletes = calls.filter((c) => c.method === "DELETE").map((c) => c.path);
		assert.deepEqual(deletes, ["/api/items/id2", "/api/items/id1"]);
		assert.ok(calls.every((c) => c.headers?.Authorization === "Bearer t"));
		await assert.rejects(scope.remove("/api/items/other"), /unowned fixture/);
	});

	it("encodes the id it deletes, so it cannot climb out of the collection", async () => {
		const { request, calls } = fake();
		(request as { post: unknown }).post = async () => ({
			ok: () => true,
			status: () => 201,
			text: async () => "",
			json: async () => ({ id: "a/../b" }),
		});
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await scope.cleanup();
		assert.deepEqual(
			calls.filter((c) => c.method === "DELETE").map((c) => c.path),
			["/api/items/a%2F..%2Fb"],
		);
	});

	it("reports a leaked record instead of swallowing it", async () => {
		const { request } = fake({ deleteStatus: () => 500 });
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await assert.rejects(scope.cleanup(), /left owned records behind/);
	});

	it("does not take a 404 on delete as proof of deletion", async () => {
		const { request } = fake({ deleteStatus: () => 404, getStatus: 200 });
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await assert.rejects(scope.cleanup(), /left owned records behind/);
	});

	it("takes a 404 on delete plus a 404 on get as gone", async () => {
		const { request } = fake({ deleteStatus: () => 404, getStatus: 404 });
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await scope.cleanup();
	});

	it("makes names unique to the scope and within 80 characters", () => {
		const a = new FixtureScope(fake().request);
		const b = new FixtureScope(fake().request);
		assert.notEqual(a.uniqueName("x"), a.uniqueName("x"));
		assert.notEqual(a.uniqueName("x"), b.uniqueName("x"));
		assert.ok(a.uniqueName("p".repeat(200)).length <= 80);
	});

	it("surfaces a created record it cannot own instead of dropping it", async () => {
		for (const body of [() => Promise.reject(new Error("bad json")), () => Promise.resolve({ id: "" })]) {
			const { request } = fake();
			(request as { post: unknown }).post = async () => ({
				ok: () => true,
				status: () => 201,
				text: async () => "",
				json: body,
			});
			const scope = new FixtureScope(request);
			await assert.rejects(scope.create("/api/items", {}), /cannot be cleaned up|no usable id/);
			// The record may exist on the server: the test must not end green.
			await assert.rejects(scope.cleanup(), /left owned records behind/);
		}
	});

	describe("retry under the hosted concurrency cap", () => {
		const posts = (calls: Call[]) => calls.filter((c) => c.method === "POST").length;

		for (const status of [503, 429]) {
			it(`retries a ${status} and returns the record once it succeeds`, async () => {
				const { request, calls } = fake({ createStatus: (n) => (n <= 2 ? status : 201) });
				const sleeps: number[] = [];
				const scope = new FixtureScope(request, {}, { sleep: async (ms) => void sleeps.push(ms) });
				const record = await scope.create<{ id: string }>("/api/items", {});
				assert.equal(record.id, "id3");
				assert.equal(posts(calls), 3);
				assert.equal(sleeps.length, 2);
				await scope.cleanup();
			});
		}

		it("gives up after the bound with the same error as before", async () => {
			const { request, calls } = fake({ createStatus: 503 });
			const scope = new FixtureScope(request, {}, { retries: 3, sleep: async () => {} });
			await assert.rejects(scope.create("/api/items", {}), /^Error: Fixture creation failed: 503 /);
			assert.equal(posts(calls), 4);
			await scope.cleanup();
		});

		it("does not retry other failures", async () => {
			for (const status of [400, 500, 502]) {
				const { request, calls } = fake({ createStatus: status });
				const scope = new FixtureScope(request, {}, { sleep: async () => {} });
				await assert.rejects(scope.create("/api/items", {}), new RegExp(`failed: ${status}`));
				assert.equal(posts(calls), 1);
			}
		});

		it("honours Retry-After, bounded by the maximum delay, and jitters within the backoff", async () => {
			const hinted = fake({ createStatus: (n) => (n === 1 ? 503 : 201), retryAfter: "2" });
			const sleeps: number[] = [];
			await new FixtureScope(hinted.request, {}, { sleep: async (ms) => void sleeps.push(ms) }).create("/api/items", {});
			assert.equal(sleeps[0], 2000);

			const capped = fake({ createStatus: (n) => (n === 1 ? 503 : 201), retryAfter: "600" });
			const capSleeps: number[] = [];
			await new FixtureScope(capped.request, {}, { maxDelayMs: 5000, sleep: async (ms) => void capSleeps.push(ms) }).create("/api/items", {});
			assert.equal(capSleeps[0], 5000);

			const plain = fake({ createStatus: (n) => (n <= 3 ? 503 : 201) });
			const waits: number[] = [];
			await new FixtureScope(plain.request, {}, { baseDelayMs: 100, maxDelayMs: 5000, sleep: async (ms) => void waits.push(ms) }).create("/api/items", {});
			waits.forEach((ms, attempt) => assert.ok(ms >= 0 && ms <= 100 * 2 ** attempt, `${ms} at attempt ${attempt}`));
		});
	});

	describe("createMany", () => {
		// Counts creates in flight and holds each for a tick so lanes overlap.
		function gauge() {
			let inFlight = 0;
			let peak = 0;
			const hold = async () => {
				inFlight += 1;
				peak = Math.max(peak, inFlight);
				await new Promise((done) => setTimeout(done, 2));
				inFlight -= 1;
			};
			return { hold, peak: () => peak };
		}

		it("never has more than the default 4 in flight, and returns records in input order", async () => {
			const g = gauge();
			const { request } = fake({ hold: g.hold });
			const scope = new FixtureScope(request);
			const records = await scope.createMany<{ id: string }>("/api/items", Array.from({ length: 20 }, (_, i) => ({ i })));
			assert.equal(records.length, 20);
			assert.equal(g.peak(), 4);
			assert.equal(new Set(records.map((r) => r.id)).size, 20);
			await scope.cleanup();
		});

		it("honours an explicit concurrency, floored at 1", async () => {
			for (const [concurrency, want] of [[2, 2], [0, 1]] as const) {
				const g = gauge();
				const scope = new FixtureScope(fake({ hold: g.hold }).request);
				await scope.createMany("/api/items", Array.from({ length: 6 }, () => ({})), { concurrency });
				assert.equal(g.peak(), want);
			}
		});

		it("returns an empty list for no items", async () => {
			const { request, calls } = fake();
			assert.deepEqual(await new FixtureScope(request).createMany("/api/items", []), []);
			assert.equal(calls.length, 0);
		});

		it("stops starting creates after a failure, throws it, and still owns what was created", async () => {
			const { request, calls } = fake({ createStatus: (n) => (n === 3 ? 400 : 201) });
			const scope = new FixtureScope(request, {}, { sleep: async () => {} });
			await assert.rejects(scope.createMany("/api/items", Array.from({ length: 30 }, () => ({})), { concurrency: 2 }), /failed: 400/);
			assert.ok(calls.filter((c) => c.method === "POST").length < 30);
			await scope.cleanup();
			assert.ok(calls.some((c) => c.method === "DELETE"));
		});
	});
});
