// NotificationBanner, rendered on the server inside the shell (__fixtures__/NotificationBannerFixture.svelte) and audited with
// axe. Delete this file and the fixture with NotificationBanner.svelte if the app has no notification banner.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import NotificationBannerFixture from './__fixtures__/NotificationBannerFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(NotificationBannerFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('NotificationBanner', () => {
	it.each([['important'], ['success']])('%s has no axe violations', async (type) => {
		mount({ type });
		expect(await audit(dom)).toEqual([]);
	});

	it('important is a region named by its title', () => {
		const banner = mount().querySelector('.govuk-notification-banner')!;
		expect(banner.getAttribute('role')).toBe('region');
		expect(banner.hasAttribute('tabindex')).toBe(false);
		const title = banner.querySelector('.govuk-notification-banner__title')!;
		expect(title.textContent?.trim()).toBe('Important');
		expect(banner.getAttribute('aria-labelledby')).toBe(title.id);
	});

	it('success is an alert that can take focus, titled Success', () => {
		const banner = mount({ type: 'success' }).querySelector('.govuk-notification-banner')!;
		expect(banner.classList.contains('govuk-notification-banner--success')).toBe(true);
		expect(banner.getAttribute('role')).toBe('alert');
		expect(banner.getAttribute('tabindex')).toBe('-1');
		expect(banner.querySelector('h2')?.textContent?.trim()).toBe('Success');
	});

	it('uses a given title, and sits above the h1', () => {
		const root = mount({ title: 'Saved' });
		const banner = root.querySelector('.govuk-notification-banner')!;
		expect(banner.querySelector('h2')?.textContent?.trim()).toBe('Saved');
		expect(banner.compareDocumentPosition(root.querySelector('h1')!) & 4).toBeTruthy();
	});
});
