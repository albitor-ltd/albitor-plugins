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
	private readonly limit: number;
	constructor(limit: number) {
		this.limit = limit;
	}
	async run<T>(job: () => Promise<T>): Promise<T> {
		if (this.active >= this.limit) {
			await new Promise<void>((resolve) => this.queue.push(resolve));
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
): Promise<T> {
	for (let n = 0; ; n++) {
		const answer = await gate.run(attempt);
		if (!isThrottled(answer.status) || n >= RETRIES) return answer;
		const wait = retryDelay(n, answer.retryAfter, random);
		if (wait === null) return answer;
		await sleep(wait);
	}
}

// Budget the app's own API reads in this context. Call it on a deployed walk only (a local app has
// no throttle). Anything that is not a same-origin /api/ GET or HEAD fetch or XHR goes as it was,
// as does a stream, because answering a call here buffers it.
export async function budgetApiReads(context: BrowserContext, origin: string, gate = new Gate(IN_FLIGHT)): Promise<void> {
	if (!origin) return;
	await context.route(`${origin}/api/**`, async (route) => {
		const req = route.request();
		const type = req.resourceType();
		const sent = req.headers();
		const read = req.method() === "GET" || req.method() === "HEAD";
		const streams = (sent.accept ?? "").includes("text/event-stream");
		if (!read || streams || (type !== "fetch" && type !== "xhr")) {
			await route.fallback();
			return;
		}
		try {
			const response = await withBudget(gate, async () => {
				const r = await route.fetch({ maxRedirects: 0 });
				return Object.assign(r, { retryAfter: r.headers()["retry-after"] });
			});
			await route.fulfill({ response });
		} catch {
			// Teardown or an outright network failure: fail the request, never throw out of a handler.
			await route.abort().catch(() => {});
		}
	});
}
