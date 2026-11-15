import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Notification } from './client';
import { consentToAsk, markAsked, shouldAsk } from './optin';

const digest: Notification = { template_id: 'weekly-digest', name: 'Weekly digest', reason: 'r', consent: false, subscribed: true };
const offers: Notification = { template_id: 'offers', name: 'Offers', reason: 'r', consent: true, subscribed: false };

// The tests run in Node, which has no localStorage: a small stand-in.
function memoryStorage(): Storage {
	const m = new Map<string, string>();
	return {
		get length() {
			return m.size;
		},
		clear: () => m.clear(),
		getItem: (k) => m.get(k) ?? null,
		key: (i) => [...m.keys()][i] ?? null,
		removeItem: (k) => void m.delete(k),
		setItem: (k, v) => void m.set(k, String(v))
	};
}

beforeEach(() => vi.stubGlobal('localStorage', memoryStorage()));
afterEach(() => vi.unstubAllGlobals());

describe('the first-sign-in opt-in', () => {
	it('asks about consent email only, never about optional email', () => {
		expect(consentToAsk([digest, offers]).map((n) => n.template_id)).toEqual(['offers']);
		expect(shouldAsk([digest], 'u-ana')).toBe(false);
	});

	it('does not ask someone who already agreed', () => {
		expect(shouldAsk([{ ...offers, subscribed: true }], 'u-ana')).toBe(false);
	});

	it('asks once per person on this browser, whatever they answered', () => {
		expect(shouldAsk([offers], 'u-ana')).toBe(true);
		markAsked('u-ana');
		expect(shouldAsk([offers], 'u-ana')).toBe(false);
		expect(shouldAsk([offers], 'u-bo')).toBe(true);
	});

	it('does not nag, or throw, when storage cannot be read', () => {
		vi.stubGlobal('localStorage', {
			getItem: () => {
				throw new Error('blocked');
			},
			setItem: () => {
				throw new Error('blocked');
			}
		});
		expect(shouldAsk([offers], 'u-ana')).toBe(false);
		expect(() => markAsked('u-ana')).not.toThrow();
	});
});
