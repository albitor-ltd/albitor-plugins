import { describe, expect, it } from 'vitest';
import { setSyncUser, getSyncUser, onSyncUser } from './session';
import { statusText } from './status';
import type { SyncStatus } from './types';

const s = (over: Partial<SyncStatus>): SyncStatus => ({
	owner: 'alice',
	pending: 0,
	failed: 0,
	syncing: false,
	needsSignIn: false,
	...over
});

describe('statusText', () => {
	it('says nothing when nobody is signed in', () => {
		expect(statusText(s({ owner: null }), true).text).toBe('');
	});

	it('puts a lapsed session first, since nothing saves until it is fixed', () => {
		const r = statusText(s({ needsSignIn: true, pending: 2, failed: 1 }), true);
		expect(r.tone).toBe('warn');
		expect(r.text).toMatch(/Sign in again/);
	});

	it('reports unsaved and rejected changes with correct plurals', () => {
		expect(statusText(s({ failed: 1 }), true).text).toMatch(/^1 change could not/);
		expect(statusText(s({ failed: 2 }), true).text).toMatch(/^2 changes could not/);
		expect(statusText(s({ pending: 1 }), true).text).toBe('1 change waiting to save.');
		expect(statusText(s({ pending: 3, syncing: true }), true).text).toBe('Saving…');
	});

	it('reassures an offline person that nothing is lost', () => {
		expect(statusText(s({ pending: 2 }), false).text).toBe('You are offline. 2 changes will save when you reconnect.');
		expect(statusText(s({}), false).text).toMatch(/on this device/);
		expect(statusText(s({}), true).text).toBe('All changes saved.');
	});
});

describe('the sync user seam', () => {
	it('tells subscribers now and on change, and ignores a repeat', () => {
		const seen: Array<string | null> = [];
		const stop = onSyncUser((u) => seen.push(u));
		setSyncUser('alice');
		setSyncUser('alice');
		setSyncUser(null);
		stop();
		setSyncUser('bob');
		expect(seen).toEqual([null, 'alice', null]);
		expect(getSyncUser()).toBe('bob');
		setSyncUser(null);
	});

	it('refuses an empty id', () => {
		expect(() => setSyncUser('')).toThrow(/non-empty/);
	});
});
