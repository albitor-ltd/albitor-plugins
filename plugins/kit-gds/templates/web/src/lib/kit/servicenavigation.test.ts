// ServiceNavigation, rendered on the server inside the shell (__fixtures__/ServiceNavigationFixture.svelte) and audited with
// axe. Delete this file and the fixture with ServiceNavigation.svelte if the app has no service navigation of its own.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import ServiceNavigationFixture from './__fixtures__/ServiceNavigationFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(ServiceNavigationFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('ServiceNavigation', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('names the service and links to its pages in a labelled navigation', () => {
		const root = mount();
		expect(root.querySelector('.govuk-service-navigation__service-name a')?.textContent).toBe('Apply for a licence');
		expect(root.querySelector('nav')?.getAttribute('aria-label')).toBe('Menu');
		expect([...root.querySelectorAll('.govuk-service-navigation__list a')].map((a) => a.getAttribute('href'))).toEqual(['/', '/reports']);
	});

	it('marks the current page, and a parent of it, and no other', () => {
		const current = (path: string) =>
			[...mount({ path }).querySelectorAll('.govuk-service-navigation__list a')].map((a) => a.getAttribute('aria-current'));
		expect(current('/reports')).toEqual([null, 'page']);
		expect(current('/reports/1')).toEqual([null, 'true']);
		expect(current('/')).toEqual(['page', null]);
	});
});
