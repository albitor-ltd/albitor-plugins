// CataloguePage in each state (CataloguePageFixture), rendered on the server the way a build prerenders it,
// loaded into a DOM and audited with axe. Delete this file and the fixture with CataloguePage.svelte.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Fixture from './__fixtures__/CataloguePageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(Fixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['catalogue', {}],
	['catalogue, filtered', { matches: 'one' }],
	['catalogue, no matches', { matches: 'none' }],
	['catalogue, first run', { matches: 'none', total: 0 }],
	['catalogue, loading', { status: 'loading' }],
	['catalogue, failed', { status: 'error' }]
];

describe('CataloguePage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('CataloguePage', () => {
	it('names the filter group and says how many results in a polite status', () => {
		const root = mount({ matches: 'one' });
		expect(root.querySelector('[role="group"][aria-label="Search and filters"] label')?.textContent).toBe(
			'Search courses'
		);
		expect(root.querySelector('p[role="status"]')?.textContent).toBe('1 of 2 results');
	});

	it('links each item by its title, once', () => {
		const root = mount();
		const links = [...root.querySelectorAll('.items h2 a')];
		expect(links.map((a) => a.textContent)).toEqual(['Pottery for beginners', 'Urban sketching']);
		expect(links[0].getAttribute('href')).toBe('/courses/pottery');
	});

	it('says nothing matched and offers to clear, in words', () => {
		const root = mount({ matches: 'none' });
		expect(root.querySelector('[data-kit="empty-state"]')?.textContent).toContain('No matches');
		expect(root.querySelector('[data-kit="empty-state"] button')?.textContent?.trim()).toBe(
			'Clear search and filters'
		);
	});

	it('says what to do first when the whole collection is empty, with no clear button', () => {
		const root = mount({ matches: 'none', total: 0 });
		expect(root.querySelector('[data-kit="empty-state"]')?.textContent).toContain('Courses are added each term.');
		expect(root.querySelector('[data-kit="empty-state"] button')).toBeNull();
	});

	it('announces loading where the results will appear and fails as an alert with a retry', () => {
		expect(mount({ status: 'loading' }).querySelector('[role="status"]')?.textContent).toContain('Loading courses');
		dom.window.close();
		const alert = mount({ status: 'error' }).querySelector('[role="alert"]');
		expect(alert?.querySelector('button')?.textContent?.trim()).toBe('Try again');
	});
});
