// Starting point (test-playwright). The walk's own request budget against a deployed app.
//
// A hosted app's function runs at most 10 calls at once and its stage takes 50 requests a second;
// over either the edge answers 429 or 503 without running the function. A browser page fires its
// calls in parallel, and a walk runs a journey after another against the same function, so an
// unlucky moment fails a step the app would have passed a second later. This file makes the walk
// behave like a well-mannered client: at most `IN_FLIGHT` of the page's own API reads at once,
// and a throttled read retried a few times with jittered backoff (honouring Retry-After up to the
// cap) before the page ever sees the 429.
//
// Reads only. A write is never repeated here: unlike fixtureScope.create, a browser write may have
// taken effect. A write the app itself must survive being throttled is the app's client's job.
import type { BrowserContext } from "@playwright/test";

export const IN_FLIGHT = 4;
export const RETRIES = 3;
export const BASE_DELAY_MS = 250;
export const MAX_DELAY_MS = 5_000;

const THROTTLED = new Set([429, 503]);

export function isThrottled(status: number): boolean {
	return THROTTLED.has(status);
}

// Full jitter over a capped exponential; a Retry-After in seconds is the floor. A hint over the cap
// means "not soon": null says give up now and let the page see the 429.
export function retryDelay(attempt: number, retryAfter: string | undefined, random: () => number = Math.random): number | null {
	const seconds = retryAfter === undefined ? NaN : Number(retryAfter);
	const hinted = Number.isFinite(seconds) && seconds >= 0 ? seconds * 1000 : 0;
	if (hinted > MAX_DELAY_MS) return null;
	return Math.max(random() * Math.min(MAX_DELAY_MS, BASE_DELAY_MS * 2 ** attempt), hinted);
}

// A counting semaphore, first come first served.
export class Gate {
	private active = 0;
	private readonly queue: Array<() => void> = [];
	// The deepest the waiting line got since `resetPeak`.
	peakQueued = 0;
	private readonly limit: number;
	constructor(limit: number) {
		this.limit = limit;
	}
	async run<T>(job: () => Promise<T>): Promise<T> {
		if (this.active >= this.limit) {
			await new Promise<void>((resolve) => {
				this.queue.push(resolve);
				this.peakQueued = Math.max(this.peakQueued, this.queue.length);
			});
		} else {
			this.active += 1;
		}
		try {
			return await job();
		} finally {
			// Hand the slot straight to the longest waiter, else free it.
			const next = this.queue.shift();
			if (next) next();
			else this.active -= 1;
		}
	}
}

type Answer = { status: number; retryAfter?: string };

// Run `attempt` until it is not throttled, at most RETRIES more times. The slot is held only while
// a call is in flight, never while waiting out a backoff.
export async function withBudget<T extends Answer>(
	gate: Gate,
	attempt: () => Promise<T>,
	sleep: (ms: number) => Promise<void> = (ms) => new Promise((done) => setTimeout(done, ms)),
	random: () => number = Math.random,
	stats?: BudgetStats,
): Promise<T> {
	stats?.read();
	for (let n = 0; ; n++) {
		const answer = await gate.run(attempt);
		if (!isThrottled(answer.status)) return answer;
		const wait = n >= RETRIES ? null : retryDelay(n, answer.retryAfter, random);
		if (wait === null) {
			stats?.gaveUp();
			return answer;
		}
		stats?.retried();
		await sleep(wait);
	}
}

// WHAT THE BUDGET HID. Retrying makes a transient throttle invisible to the page, and that same
// retry would also hide the defect a walk exists to find: an app that fires thirty reads on one
// screen, or has no client retry of its own. So every screen's cost is counted and reported, in the
// ledger line, in the test's annotations and as a finding (budgetProblem), and a read still
// throttled after its retries is a failure, exactly as it would be without the retry.
export type ScreenBudget = {
	// API reads the screen made (each counted once, however many attempts it took).
	reads: number;
	// Retries this harness made after a 429 or 503.
	retries: number;
	// Reads still throttled after the retries, which the page saw.
	throttled: number;
	// The most reads that waited for a slot at once.
	peakQueued: number;
};

// More than one throttle retry on a screen, or more than this many reads, is a finding.
export const MAX_READS_PER_SCREEN = 8;
export const MAX_RETRIES_PER_SCREEN = 1;

export class BudgetStats {
	private window: ScreenBudget = { reads: 0, retries: 0, throttled: 0, peakQueued: 0 };
	private readonly gate: Gate | undefined;
	constructor(gate?: Gate) {
		this.gate = gate;
	}
	read(): void {
		this.window.reads += 1;
	}
	retried(): void {
		this.window.retries += 1;
	}
	gaveUp(): void {
		this.window.throttled += 1;
	}
	// The screen's cost since the last take, and a fresh window.
	take(): ScreenBudget {
		const out = { ...this.window, peakQueued: this.gate?.peakQueued ?? 0 };
		this.window = { reads: 0, retries: 0, throttled: 0, peakQueued: 0 };
		if (this.gate) this.gate.peakQueued = 0;
		return out;
	}
}

// The screen's verdict. `fail` is a read the page saw throttled; `finding` is a cost the retry hid.
export function budgetProblem(b: ScreenBudget): { fail?: string; finding?: string } {
	const out: { fail?: string; finding?: string } = {};
	if (b.throttled > 0) {
		out.fail = `${b.throttled} API read(s) still throttled (429 or 503) after ${RETRIES} retries on this screen`;
	}
	const why: string[] = [];
	if (b.retries > MAX_RETRIES_PER_SCREEN) why.push(`needed ${b.retries} throttle retries`);
	if (b.reads > MAX_READS_PER_SCREEN) why.push(`fired ${b.reads} API reads (over ${MAX_READS_PER_SCREEN})`);
	if (why.length > 0) {
		out.finding = `this screen ${why.join(" and ")}: load a page's data through one aggregating endpoint`;
	}
	return out;
}

const statsOf = new WeakMap<BrowserContext, BudgetStats>();

// The context's budget counters, or undefined when it is not budgeted (a local run).
export function budgetStatsOf(context: BrowserContext): BudgetStats | undefined {
	return statsOf.get(context);
}

// Which requests the budget answers: a GET or HEAD fetch or XHR that is not a stream (answering a
// call here buffers it, which would break an event stream).
export function isBudgetedRead(method: string, accept: string | undefined, resourceType: string): boolean {
	const read = method === "GET" || method === "HEAD";
	const streams = (accept ?? "").includes("text/event-stream");
	return read && !streams && (resourceType === "fetch" || resourceType === "xhr");
}

// Budget the app's own API reads in this context. Call it on a deployed walk only (a local app has
// no throttle). Anything that is not a same-origin /api/ GET or HEAD fetch or XHR goes as it was,
// as does a stream, because answering a call here buffers it.
export async function budgetApiReads(context: BrowserContext, origin: string, gate = new Gate(IN_FLIGHT)): Promise<void> {
	if (!origin) return;
	const stats = new BudgetStats(gate);
	statsOf.set(context, stats);
	await context.route(`${origin}/api/**`, async (route) => {
		const req = route.request();
		if (!isBudgetedRead(req.method(), req.headers().accept, req.resourceType())) {
			await route.fallback();
			return;
		}
		try {
			const response = await withBudget(gate, async () => {
				const r = await route.fetch({ maxRedirects: 0 });
				return Object.assign(r, { retryAfter: r.headers()["retry-after"] });
			}, undefined, undefined, stats);
			await route.fulfill({ response });
		} catch {
			// Teardown or an outright network failure: fail the request, never throw out of a handler.
			await route.abort().catch(() => {});
		}
	});
}
