// Direct-to-bucket file transfer. The API authorises and signs; the browser then talks to the
// bucket itself, so the bytes never pass through the API function (6 MB request limit) or the
// data store.
//
// Every call answers a result and never throws on a refusal, so a screen shows what happened
// instead of an error page.
import { ApiError, NetworkError, get, post, del } from '../shell/api';

export interface FileDetails {
	id: string;
	name: string;
	content_type: string;
	size: number;
}

/**
 * A file as a list or detail screen receives it: `FileDetails` plus a same-origin `url` for `<img
 * src>` or `<a href>`, and `expires_at` (ISO time). The API puts these in the screen's own response
 * (`Files.Views`), so a list of images is one API call. Never store or log the url.
 */
export interface FileView extends FileDetails {
	url: string;
	expires_at: string;
}

/** True when the url has expired or will within `skewMs` (default 30 s): refetch the screen's data. */
export function viewExpired(view: Pick<FileView, 'expires_at'>, now: number = Date.now(), skewMs = 30_000): boolean {
	const at = Date.parse(view.expires_at);
	return Number.isNaN(at) || at - skewMs <= now;
}

export type FileResult<T = object> = ({ ok: true } & T) | { ok: false; code: string; message: string };

interface Signed {
	method: string;
	url: string;
	headers?: Record<string, string>;
	expires_at: string;
}

interface Started {
	file_id: string;
	name: string;
	upload: Signed;
}

const UNAVAILABLE = { ok: false, code: 'unavailable', message: 'File storage is unavailable: try again shortly.' } as const;
const OFFLINE = { ok: false, code: 'offline', message: 'You are offline: check your connection and try again.' } as const;

/** Refused client-side first so a person is told before any bytes move; the API checks again. */
export function checkFile(file: File, limits: { maxSize: number; types: readonly string[] }): string | null {
	if (file.size === 0) return 'That file is empty.';
	if (file.size > limits.maxSize) return `That file is larger than ${Math.round(limits.maxSize / 1024 / 1024)} MB.`;
	if (!limits.types.includes(file.type.toLowerCase())) return 'That type of file is not accepted.';
	return null;
}

async function guarded<T extends object>(run: () => Promise<T>): Promise<FileResult<T>> {
	try {
		return { ok: true, ...(await run()) };
	} catch (err) {
		if (err instanceof NetworkError) return err.offline ? OFFLINE : UNAVAILABLE;
		if (err instanceof ApiError) {
			const b = (err.body ?? {}) as { error?: unknown };
			if (err.status < 500 && typeof b.error === 'string') return { ok: false, code: String(err.status), message: b.error };
			return UNAVAILABLE;
		}
		throw err;
	}
}

/**
 * Upload one file to a record: ask the API for a signed URL, PUT the bytes to it, then tell the API
 * it is done. `complete` returns the details to keep on the record.
 *
 * Do not set Content-Length: the browser does, from the body, and the signed URL only accepts
 * exactly that length and type.
 */
export function uploadFile(record: string, file: File, signal?: AbortSignal): Promise<FileResult<{ file: FileDetails }>> {
	const base = `/records/${encodeURIComponent(record)}/files`;
	return guarded(async () => {
		const started = await post<Started>(base, { name: file.name, content_type: file.type, size: file.size });
		const { upload } = started;
		let res: Response;
		try {
			res = await fetch(upload.url, { method: upload.method, headers: upload.headers, body: file, signal });
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') throw err;
			throw new NetworkError(typeof navigator !== 'undefined' && navigator.onLine === false);
		}
		// 403 from the bucket is a signature or expiry the API would not have issued twice: ask again.
		if (!res.ok) throw new ApiError(res.status >= 500 ? res.status : 502, undefined);
		const done = await post<FileDetails>(`${base}/${started.file_id}/complete`, {});
		return { file: done };
	});
}

/** A short-lived URL to download one file; use it straight away, never store it. */
export function downloadUrl(record: string, file: string): Promise<FileResult<{ url: string }>> {
	return guarded(async () => {
		const signed = await get<Signed>(`/records/${encodeURIComponent(record)}/files/${encodeURIComponent(file)}`);
		return { url: signed.url };
	});
}

/**
 * Where a plain link can point: the API answers a redirect to a fresh signed URL.
 *
 * @deprecated Never use it for `<img src>` or any media: each one is a function call, and a list
 * of thirty images is thirty at once. Show files with the `url` in the screen's data (`FileView`).
 */
export function downloadLink(record: string, file: string): string {
	return `/api/records/${encodeURIComponent(record)}/files/${encodeURIComponent(file)}?redirect=1`;
}

export function deleteFile(record: string, file: string): Promise<FileResult> {
	return guarded(async () => {
		await del(`/records/${encodeURIComponent(record)}/files/${encodeURIComponent(file)}`);
		return {};
	});
}
