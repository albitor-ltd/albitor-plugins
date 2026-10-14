import { describe, expect, it } from 'vitest';
import { ApiError, NetworkError } from '../shell/api';
import { MAX_RETRIES, createQueryClient, retryDelay, shouldRetry } from './client';

describe('shouldRetry', () => {
	it('never retries a client error: the same request would fail the same way', () => {
		for (const status of [400, 401, 403, 404, 409, 422]) {
			expect(shouldRetry(0, new ApiError(status, null))).toBe(false);
		}
	});

	it('retries a server error, a timeout and throttling, up to the cap', () => {
		for (const status of [500, 502, 503, 408, 429]) {
			expect(shouldRetry(0, new ApiError(status, null))).toBe(true);
			expect(shouldRetry(MAX_RETRIES - 1, new ApiError(status, null))).toBe(true);
			expect(shouldRetry(MAX_RETRIES, new ApiError(status, null))).toBe(false);
		}
	});

	it('retries an unreachable API but not a browser that is offline', () => {
		expect(shouldRetry(0, new NetworkError(false))).toBe(true);
		expect(shouldRetry(0, new NetworkError(true))).toBe(false);
	});

	it('does not retry a programming error or an abort', () => {
		expect(shouldRetry(0, new TypeError('bad path'))).toBe(false);
		expect(shouldRetry(0, new DOMException('aborted', 'AbortError'))).toBe(false);
		expect(shouldRetry(0, 'boom')).toBe(false);
	});
});

describe('retryDelay', () => {
	it('backs off exponentially and is capped', () => {
		expect(retryDelay(0)).toBe(1000);
		expect(retryDelay(1)).toBe(2000);
		expect(retryDelay(2)).toBe(4000);
		expect(retryDelay(50)).toBe(8000);
	});
});

describe('createQueryClient', () => {
	it('returns an independent client each call, with the online-only defaults', () => {
		const a = createQueryClient();
		const b = createQueryClient();
		expect(a).not.toBe(b);
		const d = a.getDefaultOptions();
		expect(d.queries?.networkMode).toBe('online');
		expect(d.mutations?.networkMode).toBe('online');
		expect(d.queries?.retry).toBe(shouldRetry);
		// A mutation is not idempotent in general, so it is never retried automatically.
		expect(d.mutations?.retry).toBe(false);
	});

	it('does not retry a 404 through the real client', async () => {
		const client = createQueryClient();
		let calls = 0;
		await expect(
			client.fetchQuery({
				queryKey: ['x'],
				queryFn: () => {
					calls++;
					throw new ApiError(404, null);
				}
			})
		).rejects.toBeInstanceOf(ApiError);
		expect(calls).toBe(1);
	});
});
