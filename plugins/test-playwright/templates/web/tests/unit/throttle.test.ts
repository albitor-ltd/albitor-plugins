// Starting point (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
	BudgetStats,
	Gate,
	MAX_DELAY_MS,
	RETRIES,
	budgetApiReads,
	budgetProblem,
	isBudgetedRead,
	retryDelay,
	withBudget,
	budgetStatsOf,
} from "../helpers/throttle.ts";

const noSleep = async () => {};

describe("retryDelay", () => {
	it("jitters under a capped exponential ceiling", () => {
		assert.equal(retryDelay(0, undefined, () => 1), 250);
		assert.equal(retryDelay(1, undefined, () => 1), 500);
		assert.equal(retryDelay(10, undefined, () => 1), MAX_DELAY_MS);
		assert.equal(retryDelay(0, undefined, () => 0), 0);
	});
	it("never retries sooner than Retry-After, and gives up past the cap", () => {
		assert.equal(retryDelay(0, "2", () => 0), 2000);
		assert.equal(retryDelay(0, "6", () => 0), null);
		assert.equal(retryDelay(0, "soon", () => 0), 0);
	});
});

describe("withBudget", () => {
	it("retries a throttled read until it is answered", async () => {
		let calls = 0;
		const out = await withBudget(new Gate(4), async () => ({ status: ++calls < 3 ? 429 : 200 }), noSleep, () => 0);
		assert.equal(out.status, 200);
		assert.equal(calls, 3);
	});
	it("returns the last throttled answer after the retries are spent", async () => {
		let calls = 0;
		const out = await withBudget(new Gate(4), async () => ({ status: (calls++, 503) }), noSleep, () => 0);
		assert.equal(out.status, 503);
		assert.equal(calls, RETRIES + 1);
	});
	it("does not retry an ordinary failure, and does not wait out a long Retry-After", async () => {
		let calls = 0;
		assert.equal((await withBudget(new Gate(4), async () => ({ status: (calls++, 500) }), noSleep)).status, 500);
		assert.equal(calls, 1);
		calls = 0;
		await withBudget(new Gate(4), async () => ({ status: (calls++, 429), retryAfter: "60" }), noSleep);
		assert.equal(calls, 1);
	});
	it("holds no slot while it backs off", async () => {
		const gate = new Gate(1);
		let release!: () => void;
		const slept = new Promise<void>((r) => (release = r));
		let first = 0;
		const a = withBudget(gate, async () => ({ status: ++first === 1 ? 429 : 200 }), () => slept, () => 0);
		await new Promise((r) => setTimeout(r, 5));
		// The only slot is free while `a` sleeps, so another call runs at once.
		assert.equal(await gate.run(async () => "ran"), "ran");
		release();
		assert.equal((await a).status, 200);
	});
});

describe("Gate", () => {
	it("never runs more than its limit at once, and runs everything", async () => {
		const gate = new Gate(4);
		let active = 0;
		let peak = 0;
		await Promise.all(
			Array.from({ length: 20 }, () =>
				gate.run(async () => {
					peak = Math.max(peak, ++active);
					await new Promise((r) => setTimeout(r, 2));
					active--;
				}),
			),
		);
		assert.equal(peak, 4);
		assert.equal(active, 0);
	});
	it("frees the slot when a job throws", async () => {
		const gate = new Gate(1);
		await assert.rejects(gate.run(async () => { throw new Error("x"); }));
		assert.equal(await gate.run(async () => 1), 1);
	});
});

describe("isBudgetedRead", () => {
	it("answers GET and HEAD fetch and XHR reads", () => {
		assert.equal(isBudgetedRead("GET", "application/json", "fetch"), true);
		assert.equal(isBudgetedRead("HEAD", undefined, "xhr"), true);
	});
	it("leaves writes, streams and every other resource alone", () => {
		assert.equal(isBudgetedRead("POST", "application/json", "fetch"), false);
		assert.equal(isBudgetedRead("DELETE", undefined, "fetch"), false);
		assert.equal(isBudgetedRead("GET", "text/event-stream", "fetch"), false);
		assert.equal(isBudgetedRead("GET", "text/html", "document"), false);
		assert.equal(isBudgetedRead("GET", "*/*", "script"), false);
	});
});

describe("budgetApiReads routing", () => {
	// A fake context that records the one handler it is given.
	function fakeContext() {
		let handler!: (route: unknown) => Promise<void>;
		const context = {
			route: async (_glob: string, h: typeof handler) => {
				handler = h;
			},
		};
		return { context, run: (route: unknown) => handler(route) };
	}
	function fakeRoute(method: string, resourceType: string, statuses: number[], accept = "application/json") {
		const log = { fallback: 0, fetched: 0, fulfilled: 0 };
		const route = {
			request: () => ({ method: () => method, resourceType: () => resourceType, headers: () => ({ accept }) }),
			fallback: async () => void log.fallback++,
			fetch: async () => {
				const status = statuses[Math.min(log.fetched++, statuses.length - 1)];
				return { status: () => status, headers: () => ({}) , status_: status };
			},
			fulfill: async () => void log.fulfilled++,
			abort: async () => {},
		};
		return { route, log };
	}

	it("falls back for a write, a stream and a page, and answers a read", async () => {
		const { context, run } = fakeContext();
		await budgetApiReads(context as never, "https://app.example");
		for (const [m, t, a] of [["POST", "fetch", "application/json"], ["GET", "fetch", "text/event-stream"], ["GET", "document", "text/html"]] as const) {
			const { route, log } = fakeRoute(m, t, [200], a);
			await run(route);
			assert.deepEqual(log, { fallback: 1, fetched: 0, fulfilled: 0 });
		}
		const { route, log } = fakeRoute("GET", "fetch", [200]);
		await run(route);
		assert.deepEqual(log, { fallback: 0, fetched: 1, fulfilled: 1 });
		assert.equal(budgetStatsOf(context as never)?.take().reads, 1);
	});
});

describe("what the budget reports", () => {
	it("counts reads, retries and a read that stayed throttled, and resets per screen", async () => {
		const stats = new BudgetStats();
		let n = 0;
		await withBudget(new Gate(4), async () => ({ status: ++n < 3 ? 429 : 200 }), noSleep, () => 0, stats);
		await withBudget(new Gate(4), async () => ({ status: 503 }), noSleep, () => 0, stats);
		assert.deepEqual(stats.take(), { reads: 2, retries: 2 + RETRIES, throttled: 1, peakQueued: 0 });
		assert.deepEqual(stats.take(), { reads: 0, retries: 0, throttled: 0, peakQueued: 0 });
	});
	it("records the deepest queue", async () => {
		const gate = new Gate(1);
		const stats = new BudgetStats(gate);
		await Promise.all(Array.from({ length: 4 }, () => gate.run(() => new Promise((r) => setTimeout(r, 2)))));
		assert.equal(stats.take().peakQueued, 3);
		assert.equal(stats.take().peakQueued, 0);
	});
	it("fails a screen whose read stayed throttled, and flags a hidden cost without failing it", () => {
		assert.deepEqual(budgetProblem({ reads: 3, retries: 1, throttled: 0, peakQueued: 0 }), {});
		assert.match(budgetProblem({ reads: 3, retries: 4, throttled: 1, peakQueued: 0 }).fail ?? "", /still throttled/);
		assert.match(budgetProblem({ reads: 3, retries: 2, throttled: 0, peakQueued: 0 }).finding ?? "", /2 throttle retries/);
		const many = budgetProblem({ reads: 9, retries: 0, throttled: 0, peakQueued: 5 });
		assert.equal(many.fail, undefined);
		assert.match(many.finding ?? "", /9 API reads.*one aggregating endpoint/);
	});
});
