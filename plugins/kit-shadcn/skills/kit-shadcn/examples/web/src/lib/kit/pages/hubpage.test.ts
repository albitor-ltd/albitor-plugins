// HubPage in each state (HubPageFixture), rendered on the server the way a build prerenders it,
// loaded into a DOM and audited with axe. Delete this file and the fixture with HubPage.svelte and BackToHub.svelte.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Fixture from './__fixtures__/HubPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(Fixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['hub', {}],
	['hub, loading', { status: 'loading' }],
	['hub, failed', { status: 'error' }],
	['inner page with the back-to-hub link', { page: 'inner' }],
	['inner page with the header\'s back link', { page: 'inner-header' }]
];

describe('HubPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('HubPage', () => {
	it('with no nav there is no empty Main nav landmark, only the skip link, header, main and toaster', () => {
		const root = mount();
		expect(root.querySelector('nav')).toBeNull();
		expect(root.querySelector('main')).not.toBeNull();
		expect(root.querySelector('header')).not.toBeNull();
	});

	it('links every destination from a tile, by its title, and says where each stands in words', () => {
		const root = mount();
		expect([...root.querySelectorAll('.tiles h3 a')].map((a) => a.getAttribute('href'))).toEqual([
			'/renew',
			'/letters'
		]);
		expect(root.querySelector('.tiles [data-kit="pill"]')?.textContent).toContain('Due 1 December');
	});
});

describe('the way back to the hub', () => {
	it('is a visible link to the hub at the top of an inner page', () => {
		const root = mount({ page: 'inner' });
		const link = root.querySelector('main a');
		expect(link?.getAttribute('href')).toBe('/');
		expect(link?.textContent?.trim()).toBe('Back to your account');
		expect(link?.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true');
	});

	it('is the same link when the page passes it as the header\'s back', () => {
		const link = mount({ page: 'inner-header' }).querySelector('main .back a');
		expect(link?.getAttribute('href')).toBe('/');
		expect(link?.textContent?.trim()).toBe('Back to your account');
	});
});
