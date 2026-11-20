// HomePage in each state (HomePageFixture), rendered on the server the way a build prerenders it,
// loaded into a DOM and audited with axe. Delete this file and the fixture with HomePage.svelte.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Fixture from './__fixtures__/HomePageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(Fixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['home', {}],
	['home, headline and action only', { bare: true }]
];

describe('HomePage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('HomePage', () => {
	it('puts the headline in the h1 and the primary action in the first link of the page', () => {
		const root = mount();
		expect(root.querySelector('h1')?.textContent).toBe('Book a pitch at the club');
		expect(root.querySelector('main a')?.textContent?.trim()).toBe('See free pitches');
	});

	it('draws the reasons as a list, with decorative icons', () => {
		const root = mount();
		expect(root.querySelectorAll('.highlights li')).toHaveLength(2);
		expect(root.querySelector('.highlights svg')?.getAttribute('aria-hidden')).toBe('true');
	});

	it('uses no images', () => {
		expect(mount().querySelector('main img')).toBeNull();
	});
});
