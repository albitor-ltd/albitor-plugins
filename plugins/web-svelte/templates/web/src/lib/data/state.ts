// Starting point (web-svelte, online data layer): which of the five states a query is in, and what to tell the
// person about a failure.
import { ApiError, NetworkError } from '../shell/api';

export type ViewState = 'loading' | 'offline' | 'error' | 'empty' | 'ready';

/** The part of a query result `viewState` reads, so any `createQuery` result satisfies it. */
export interface QueryLike {
	status: 'pending' | 'error' | 'success';
	fetchStatus: 'fetching' | 'paused' | 'idle';
	data: unknown;
}

/** Nothing, an empty array, or the API's list envelope (`{ items: [] }`) with no items. */
export function isEmptyData(data: unknown): boolean {
	if (data === undefined || data === null) return true;
	if (Array.isArray(data)) return data.length === 0;
	if (typeof data === 'object' && 'items' in data && Array.isArray(data.items)) {
		return data.items.length === 0;
	}
	return false;
}

/**
 * A failed refetch that still has data shows that data: the person keeps working, and the next
 * refetch tries again. Only a query with nothing to show reaches 'error'.
 */
export function viewState(q: QueryLike, isEmpty: (data: unknown) => boolean = isEmptyData): ViewState {
	if (q.status === 'success') return isEmpty(q.data) ? 'empty' : 'ready';
	if (q.status === 'error') return q.data !== undefined ? 'ready' : 'error';
	return q.fetchStatus === 'paused' ? 'offline' : 'loading';
}

/**
 * Plain words for a failure. It never repeats the response body or an error's own message: both
 * can carry server internals, and neither belongs on screen.
 */
export function errorMessage(error: unknown): string {
	if (error instanceof NetworkError) {
		return error.offline
			? 'You are offline. This will load when you are back online.'
			: 'The service could not be reached. Try again.';
	}
	if (error instanceof ApiError) {
		if (error.status === 401) return 'Your session has ended. Sign in again.';
		if (error.status === 403) return 'You do not have permission to see this.';
		if (error.status === 404) return 'This was not found.';
		return 'The service had a problem. Try again.';
	}
	return 'Something went wrong. Try again.';
}
