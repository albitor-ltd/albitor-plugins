import { afterEach, describe, expect, it, vi } from 'vitest';
import { changedChoices, hasConsentEmail, loadNotifications, saveNotifications, type Notification } from './client';

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

const digest: Notification = { template_id: 'weekly-digest', name: 'Weekly digest', reason: 'What changed this week.', consent: false, subscribed: true };
const offers: Notification = { template_id: 'offers', name: 'Offers', reason: 'News, only if you agree.', consent: true, subscribed: false };

afterEach(() => vi.unstubAllGlobals());

describe('loadNotifications', () => {
	it('returns the emails with their choices', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => json(200, { notifications: [digest, offers] })));
		expect(await loadNotifications()).toEqual({ ok: true, notifications: [digest, offers] });
	});

	it('says the settings are unavailable, never an error page or a code, when the API cannot answer', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => json(503, { error: 'tenant_paused' })));
		const out = await loadNotifications();
		expect(out).toEqual({ ok: false, message: expect.stringMatching(/unavailable/i) });
		expect(JSON.stringify(out)).not.toMatch(/tenant_paused/);
	});
});

describe('saveNotifications', () => {
	it('puts only the choices it is given', async () => {
		const fetch = vi.fn(async () => new Response(null, { status: 204 }));
		vi.stubGlobal('fetch', fetch);
		expect(await saveNotifications({ offers: true })).toEqual({ ok: true });
		const [url, init] = fetch.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('/api/notifications');
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body as string)).toEqual({ choices: { offers: true } });
	});

	it('sends nothing when nothing changed', async () => {
		const fetch = vi.fn();
		vi.stubGlobal('fetch', fetch);
		expect(await saveNotifications({})).toEqual({ ok: true });
		expect(fetch).not.toHaveBeenCalled();
	});

	it('shows the API refusal in its own words', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => json(422, { error: 'one of those emails is not a choice' })));
		expect(await saveNotifications({ x: true })).toEqual({ ok: false, message: 'one of those emails is not a choice' });
	});
});

describe('changedChoices', () => {
	it('keeps only what differs from what was loaded, so an untouched consent switch is never sent as an opt-in', () => {
		expect(changedChoices([digest, offers], { 'weekly-digest': true, offers: false })).toEqual({});
		expect(changedChoices([digest, offers], { 'weekly-digest': false, offers: true })).toEqual({ 'weekly-digest': false, offers: true });
	});

	it('ignores an id that was not loaded', () => {
		expect(changedChoices([digest], { stranger: true })).toEqual({});
	});
});

describe('hasConsentEmail', () => {
	it('is true only when some email needs the person to agree', () => {
		expect(hasConsentEmail([digest])).toBe(false);
		expect(hasConsentEmail([digest, offers])).toBe(true);
	});
});
