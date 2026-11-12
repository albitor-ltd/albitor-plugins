// Starting point (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Gate, MAX_DELAY_MS, RETRIES, retryDelay, withBudget } from "../helpers/throttle.ts";

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
