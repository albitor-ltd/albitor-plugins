// CollectionPage in each state (CollectionPageFixture), rendered on the server the way a build prerenders it,
// loaded into a DOM and audited with axe. Delete this file and the fixture with CollectionPage.svelte.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Fixture from './__fixtures__/CollectionPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(Fixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['collection', {}],
	['collection, first run', { empty: true }],
	['collection, loading', { status: 'loading' }],
	['collection, failed', { status: 'error' }]
];

describe('CollectionPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('CollectionPage', () => {
	it('shows the one-line summary and links each item by its name', () => {
		const root = mount();
		expect(root.querySelector('.summary')?.textContent).toBe('2 plants, 1 needs water');
		expect([...root.querySelectorAll('.grid h2 a')].map((a) => a.textContent)).toEqual(['Boston fern', 'Basil']);
	});

	it('says an item needs attention in words, not colour alone', () => {
		expect(mount().querySelector('[data-kit="pill"]')?.textContent).toContain('Needs water');
	});

	it('says what to do first when empty, with an icon hidden from assistive technology', () => {
		const root = mount({ empty: true });
		expect(root.querySelector('[data-kit="empty-state"]')?.textContent).toContain('Add the first one');
		expect(root.querySelector('[data-kit="empty-state"] button')?.textContent?.trim()).toBe('Add a plant');
		expect(root.querySelector('[data-kit="empty-state"] svg')?.getAttribute('aria-hidden')).toBe('true');
	});
});
