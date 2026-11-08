// The API client runtime. Every call goes to this app's own
// /api, which the host (or `npm run local`, or the dev server) forwards to the API.
//
// It is a file, not a package, until a published client exists; keep app code behind these
// functions so that swap stays a one-file change.

export const API_BASE = '/api';

/** The API answered with a non-2xx status. */
export class ApiError extends Error {
	/**
	 * @param retried true when `api` already applied its retry policy to this failure (a throttled
	 * read that was retried or whose `Retry-After` was too long to wait), so a caller must not retry it again.
	 */
	constructor(
		readonly status: number,
		readonly body: unknown,
		readonly retried = false
	) {
		super(`API request failed with status ${status}`);
		this.name = 'ApiError';
	}
}

/** The request never reached the API. `offline` is true when the browser reports no network. */
export class NetworkError extends Error {
	/** `retried` is as on {@link ApiError}. */
	constructor(
		readonly offline: boolean,
		readonly retried = false
	) {
		super(offline ? 'You are offline' : 'The API could not be reached');
		this.name = 'NetworkError';
	}
}

type TokenSource = () => string | null | undefined | Promise<string | null | undefined>;

let tokenSource: TokenSource | null = null;

/** The auth plugin's provider calls this once so every request carries the session token. */
export function setTokenSource(source: TokenSource | null): void {
	tokenSource = source;
}

export interface RequestOptions {
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	/** Serialised as JSON. */
	body?: unknown;
	query?: Record<string, string | number | boolean | undefined>;
	signal?: AbortSignal;
}

/** A path is one app-relative API path: no scheme, host, query, fragment, dot segments
 * or encoded dots, slashes and backslashes. */
function checkPath(path: string): void {
	const segments = path.split('/');
	if (
		!/^\/(?!\/)[A-Za-z0-9._~%!$&'()*+,;=:@/-]*$/.test(path) ||
		// Encoded dots, slashes and backslashes are refused outright (any case), because a server
		// or proxy may decode them into a dot segment after this check.
		/%(2e|2f|5c)/i.test(path) ||
		segments.includes('..') ||
		segments.includes('.')
	) {
		throw new TypeError(`API path ${JSON.stringify(path)} must be an app-relative path such as /items`);
	}
}

/** Most requests in flight at once per tab. The host allows an app only a few concurrent
 * executions, so a burst beyond this would be throttled; the rest wait their turn here. */
export const MAX_IN_FLIGHT = 6;

/** Attempts in all (the first plus retries) for a GET or HEAD that is throttled or unreachable. */
export const MAX_ATTEMPTS = 3;
/** First backoff ceiling; it doubles each retry. */
export const BASE_DELAY_MS = 250;
/** Longest backoff, and the longest `Retry-After` that is waited for: a longer one fails at once. */
export const MAX_DELAY_MS = 5000;

const abortError = () => new DOMException('The request was aborted', 'AbortError');
const isAbort = (err: unknown) => err instanceof DOMException && err.name === 'AbortError';

let inFlight = 0;
const waiting: Array<() => void> = [];

/** Takes a slot, first come first served. A queued request whose signal aborts leaves the queue. */
function acquire(signal?: AbortSignal): Promise<void> {
	if (signal?.aborted) return Promise.reject(abortError());
	if (inFlight < MAX_IN_FLIGHT) {
		inFlight++;
		return Promise.resolve();
	}
	return new Promise<void>((resolve, reject) => {
		const grant = () => {
			signal?.removeEventListener('abort', onAbort);
			resolve();
		};
		const onAbort = () => {
			const i = waiting.indexOf(grant);
			if (i >= 0) waiting.splice(i, 1);
			reject(abortError());
		};
		waiting.push(grant);
		signal?.addEventListener('abort', onAbort, { once: true });
	});
}

/** Frees a slot, handing it straight to the longest-waiting request. */
function release(): void {
	const next = waiting.shift();
	if (next) next();
	else inFlight--;
}

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
	if (signal?.aborted) return Promise.reject(abortError());
	return new Promise<void>((resolve, reject) => {
		const onAbort = () => {
			clearTimeout(timer);
			reject(abortError());
		};
		const timer = setTimeout(() => {
			signal?.removeEventListener('abort', onAbort);
			resolve();
		}, ms);
		signal?.addEventListener('abort', onAbort, { once: true });
	});
}

/** `Retry-After` as milliseconds: delay-seconds or an HTTP date. Null when absent or unreadable. */
function retryAfterMs(res: Response): number | null {
	const value = res.headers.get('Retry-After')?.trim();
	if (!value) return null;
	if (/^\d+$/.test(value)) return Number(value) * 1000;
	const at = Date.parse(value);
	return Number.isNaN(at) ? null : Math.max(0, at - Date.now());
}

/** Full jitter over a capped exponential, never sooner than the (already capped) `Retry-After`. */
function backoff(attempt: number, retryAfter: number | null): number {
	const ceiling = Math.min(MAX_DELAY_MS, BASE_DELAY_MS * 2 ** attempt);
	return Math.max(Math.random() * ceiling, retryAfter ?? 0);
}

export async function api<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
	checkPath(path);
	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(options.query ?? {})) {
		if (value !== undefined) params.set(key, String(value));
	}
	const search = params.size > 0 ? `?${params}` : '';

	let body: string | undefined;
	if (options.body !== undefined) body = JSON.stringify(options.body);
	const method = options.method ?? (body === undefined ? 'GET' : 'POST');
	// Only a read is safe to repeat. Never retry a write: it may have taken effect.
	const idempotent = method === 'GET' || (method as string) === 'HEAD';
	const maxAttempts = idempotent ? MAX_ATTEMPTS : 1;

	for (let attempt = 0; ; attempt++) {
		const last = attempt + 1 >= maxAttempts;
		let wait: number | null = null;
		await acquire(options.signal);
		try {
			const headers: Record<string, string> = { Accept: 'application/json' };
			// Read on every attempt, so a retry carries a token refreshed in the meantime.
			const token = tokenSource ? await tokenSource() : null;
			if (token) headers.Authorization = `Bearer ${token}`;
			if (body !== undefined) headers['Content-Type'] = 'application/json';

			let res: Response;
			try {
				res = await fetch(`${API_BASE}${path}${search}`, {
					method,
					headers,
					body,
					signal: options.signal
				});
			} catch (err) {
				if (isAbort(err)) throw err;
				const offline = typeof navigator !== 'undefined' && navigator.onLine === false;
				if (!idempotent || offline) throw new NetworkError(offline);
				if (last) throw new NetworkError(false, true);
				wait = backoff(attempt, null);
				continue;
			}

			if (idempotent && (res.status === 429 || res.status === 503)) {
				const after = retryAfterMs(res);
				const payload = await readBody(res);
				// A wait over the cap means "not soon": give up now rather than retry early.
				if (last || (after !== null && after > MAX_DELAY_MS)) {
					throw new ApiError(res.status, payload, true);
				}
				wait = backoff(attempt, after);
				continue;
			}

			const payload = await readBody(res);
			if (!res.ok) throw new ApiError(res.status, payload);
			return payload as T;
		} finally {
			// Released before any backoff wait, so waiting never holds a slot.
			release();
			if (wait !== null) await sleep(wait, options.signal);
		}
	}
}

async function readBody(res: Response): Promise<unknown> {
	if (res.status === 204 || res.status === 205) return undefined;
	const text = await res.text();
	if (text === '') return undefined;
	if ((res.headers.get('Content-Type') ?? '').includes('json')) {
		try {
			return JSON.parse(text);
		} catch {
			return text;
		}
	}
	return text;
}

export const get = <T = unknown>(path: string, query?: RequestOptions['query']) =>
	api<T>(path, { query });
export const post = <T = unknown>(path: string, body?: unknown) =>
	api<T>(path, { method: 'POST', body });
export const put = <T = unknown>(path: string, body?: unknown) =>
	api<T>(path, { method: 'PUT', body });
export const del = <T = unknown>(path: string) => api<T>(path, { method: 'DELETE' });
