// Managed by Albitor (web-svelte): the API client runtime. Every call goes to this app's own
// /api, which the host (or `npm run local`, or the dev server) forwards to the API.
//
// It is a file, not a package, until a published client exists; keep app code behind these
// functions so that swap stays a one-file change.

export const API_BASE = '/api';

/** The API answered with a non-2xx status. */
export class ApiError extends Error {
	constructor(
		readonly status: number,
		readonly body: unknown
	) {
		super(`API request failed with status ${status}`);
		this.name = 'ApiError';
	}
}

/** The request never reached the API. `offline` is true when the browser reports no network. */
export class NetworkError extends Error {
	constructor(readonly offline: boolean) {
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

export async function api<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
	checkPath(path);
	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(options.query ?? {})) {
		if (value !== undefined) params.set(key, String(value));
	}
	const search = params.size > 0 ? `?${params}` : '';

	const headers: Record<string, string> = { Accept: 'application/json' };
	const token = tokenSource ? await tokenSource() : null;
	if (token) headers.Authorization = `Bearer ${token}`;
	let body: string | undefined;
	if (options.body !== undefined) {
		headers['Content-Type'] = 'application/json';
		body = JSON.stringify(options.body);
	}

	let res: Response;
	try {
		res = await fetch(`${API_BASE}${path}${search}`, {
			method: options.method ?? (body === undefined ? 'GET' : 'POST'),
			headers,
			body,
			signal: options.signal
		});
	} catch (err) {
		if (err instanceof DOMException && err.name === 'AbortError') throw err;
		throw new NetworkError(typeof navigator !== 'undefined' && navigator.onLine === false);
	}

	const payload = await readBody(res);
	if (!res.ok) throw new ApiError(res.status, payload);
	return payload as T;
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
