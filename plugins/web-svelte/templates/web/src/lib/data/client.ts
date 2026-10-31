// Starting point (web-svelte, online data layer): the query client every page shares.
//
// Online-only: nothing is persisted, and a query waits (it does not fail) while the browser has no
// network. The retry policy is here so that every query in the app behaves the same way.
import { QueryClient } from '@tanstack/svelte-query';
import { ApiError, NetworkError } from '../shell/api';

/** Automatic retries after the first attempt. */
export const MAX_RETRIES = 2;

/**
 * Retry only what could plausibly succeed next time: a server error, a timeout, throttling or an
 * unreachable API. A client error (400 to 499 apart from 408 and 429) would fail identically, an
 * offline browser is handled by pausing, and anything else is a bug that retrying hides.
 */
export function shouldRetry(failureCount: number, error: unknown): boolean {
	if (failureCount >= MAX_RETRIES) return false;
	if (error instanceof ApiError) {
		return error.status >= 500 || error.status === 408 || error.status === 429;
	}
	if (error instanceof NetworkError) return !error.offline;
	return false;
}

/** 1s, 2s, 4s, then 8s at most. `attempt` counts from 0. */
export function retryDelay(attempt: number): number {
	return Math.min(1000 * 2 ** Math.max(0, attempt), 8000);
}

export function createQueryClient(): QueryClient {
	return new QueryClient({
		defaultOptions: {
			queries: {
				retry: shouldRetry,
				retryDelay,
				// Data under 30 seconds old is served from the cache without a request.
				staleTime: 30_000,
				gcTime: 5 * 60_000,
				refetchOnWindowFocus: true,
				networkMode: 'online'
			},
			// A mutation is not idempotent in general, so it is never retried automatically.
			mutations: { retry: false, networkMode: 'online' }
		}
	});
}
