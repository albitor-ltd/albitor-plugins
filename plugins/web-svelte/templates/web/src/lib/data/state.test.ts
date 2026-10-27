import { describe, expect, it } from 'vitest';
import { ApiError, NetworkError } from '../shell/api';
import { errorMessage, isEmptyData, viewState, type QueryLike } from './state';

const q = (over: Partial<QueryLike>): QueryLike => ({
	status: 'pending',
	fetchStatus: 'fetching',
	data: undefined,
	...over
});

describe('viewState', () => {
	it('is loading while the first fetch is in flight', () => {
		expect(viewState(q({}))).toBe('loading');
	});

	it('is offline when the first fetch is paused for want of a network', () => {
		expect(viewState(q({ fetchStatus: 'paused' }))).toBe('offline');
	});

	it('is error when the fetch failed and there is nothing to show', () => {
		expect(viewState(q({ status: 'error', fetchStatus: 'idle' }))).toBe('error');
	});

	it('keeps showing data it already has when a refetch fails', () => {
		expect(viewState(q({ status: 'error', fetchStatus: 'idle', data: [1] }))).toBe('ready');
	});

	it('is empty for an empty success and ready for a populated one', () => {
		expect(viewState(q({ status: 'success', fetchStatus: 'idle', data: [] }))).toBe('empty');
		expect(viewState(q({ status: 'success', fetchStatus: 'idle', data: [1] }))).toBe('ready');
	});

	it('takes the caller own emptiness test', () => {
		const result = q({ status: 'success', fetchStatus: 'idle', data: { total: 0 } });
		expect(viewState(result)).toBe('ready');
		expect(viewState(result, (d) => (d as { total: number }).total === 0)).toBe('empty');
	});
});

describe('isEmptyData', () => {
	it('treats nothing, an empty array and an empty page envelope as empty', () => {
		expect(isEmptyData(undefined)).toBe(true);
		expect(isEmptyData(null)).toBe(true);
		expect(isEmptyData([])).toBe(true);
		expect(isEmptyData({ items: [] })).toBe(true);
	});

	it('treats everything with content as not empty', () => {
		expect(isEmptyData([0])).toBe(false);
		expect(isEmptyData({ items: [1] })).toBe(false);
		expect(isEmptyData({})).toBe(false);
		expect(isEmptyData(0)).toBe(false);
		expect(isEmptyData('')).toBe(false);
		expect(isEmptyData(false)).toBe(false);
	});
});

describe('errorMessage', () => {
	it('explains an offline browser and an unreachable API differently', () => {
		expect(errorMessage(new NetworkError(true))).toMatch(/offline/i);
		expect(errorMessage(new NetworkError(false))).toMatch(/could not be reached/i);
	});

	it('maps API statuses to plain words', () => {
		expect(errorMessage(new ApiError(401, null))).toMatch(/sign in/i);
		expect(errorMessage(new ApiError(403, null))).toMatch(/permission/i);
		expect(errorMessage(new ApiError(404, null))).toMatch(/not found/i);
		expect(errorMessage(new ApiError(503, null))).toMatch(/try again/i);
	});

	it('never echoes the response body or a raw error message', () => {
		const raw = 'stack trace /srv/app/internal.go:12 detail=abc';
		expect(errorMessage(new ApiError(500, { error: raw }))).not.toContain('internal');
		expect(errorMessage(new Error(raw))).not.toContain('internal');
		expect(errorMessage('x')).toBe('Something went wrong. Try again.');
	});
});
