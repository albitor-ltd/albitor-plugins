import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { ApiError } from '../shell/api';
import DataState from './DataState.svelte';
import type { QueryLike } from './state';

const body = createRawSnippet((data: () => unknown) => ({
	render: () => `<p>ready:${JSON.stringify(data())}</p>`
}));

const q = (over: Partial<QueryLike> & { error?: unknown; refetch?: () => unknown }) => ({
	status: 'pending' as const,
	fetchStatus: 'fetching' as const,
	data: undefined,
	...over
});

const html = (query: ReturnType<typeof q>, extra: Record<string, unknown> = {}) =>
	render(DataState, { props: { query, children: body, ...extra } }).body;

describe('DataState', () => {
	it('announces loading', () => {
		expect(html(q({}))).toMatch(/role="status"[^>]*>Loading/);
	});

	it('announces offline when the first fetch is paused', () => {
		expect(html(q({ fetchStatus: 'paused' }))).toMatch(/role="status"[^>]*>You are offline/);
	});

	it('shows an alert with plain wording and Try again, without the response body', () => {
		const out = html(
			q({
				status: 'error',
				fetchStatus: 'idle',
				error: new ApiError(500, { error: 'secret internals' }),
				refetch: () => {}
			})
		);
		expect(out).toMatch(/role="alert"/);
		expect(out).toContain('Try again');
		expect(out).not.toContain('secret');
	});

	it('offers no retry button when the query cannot refetch', () => {
		expect(html(q({ status: 'error', fetchStatus: 'idle', error: new Error('x') }))).not.toContain('<button');
	});

	it('shows the empty text for an empty success, and the caller text when given', () => {
		const empty = q({ status: 'success', fetchStatus: 'idle', data: { items: [] } });
		expect(html(empty)).toContain('Nothing here yet.');
		expect(html(empty, { emptyText: 'No invoices.' })).toContain('No invoices.');
	});

	it('renders the data when ready', () => {
		expect(html(q({ status: 'success', fetchStatus: 'idle', data: [1] }))).toContain('ready:[1]');
	});
});
