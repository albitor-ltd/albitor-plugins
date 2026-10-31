// Starting point (web-svelte, online data layer): query options for this app's own API.
import { queryOptions } from '@tanstack/svelte-query';
import { api, type RequestOptions } from '../shell/api';

/**
 * Every API query key starts with this, so `queryClient.invalidateQueries({ queryKey: [API_KEY] })`
 * refreshes all of them and `queryClient.clear()` on sign-out drops the lot.
 */
export const API_KEY = 'api';

/**
 * Options for `createQuery(() => apiQuery('/items', { page: 1 }))`. The request goes through the
 * shell's API client, so the path is checked and the bearer token is attached there, and the
 * query's abort signal cancels the request.
 */
export function apiQuery<T = unknown>(path: string, query?: RequestOptions['query']) {
	return queryOptions({
		queryKey: [API_KEY, path, query ?? {}] as const,
		queryFn: ({ signal }) => api<T>(path, { query, signal })
	});
}
