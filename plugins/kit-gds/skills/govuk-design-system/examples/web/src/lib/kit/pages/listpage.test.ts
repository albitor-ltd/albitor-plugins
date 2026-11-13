// ListPage in each state (__fixtures__/ListPageFixture.svelte), rendered on the server the way a
// build prerenders it, loaded into a DOM and audited with axe. Delete this file and the fixture
// with ListPage.svelte if the app has no such page.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import ListPageFixture from './__fixtures__/ListPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(ListPageFixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['list', {}],
	['list, loading', { status: 'loading' }],
	['list, failed', { status: 'error' }],
	['list, empty', { rows: [] }]
];

describe('ListPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('states', () => {
	it('announces loading politely, where the content will appear', () => {
		const root = mount({ status: 'loading' });
		expect(root.querySelector('[role="status"]')?.textContent).toContain('Loading reports');
		expect(root.querySelector('table')).toBeNull();
	});

	it('shows a failure as an alert with a retry button', () => {
		const root = mount({ status: 'error' });
		const alert = root.querySelector('.govuk-notification-banner[role="alert"]');
		expect(alert?.textContent).toContain('Something went wrong');
		expect(alert?.querySelector('button')?.textContent?.trim()).toBe('Try again');
	});

	it('says what to do next when a list is empty', () => {
		const root = mount({ rows: [] });
		expect(root.querySelector('main')?.textContent).toContain('Create the first report');
		expect(root.querySelector('table')).toBeNull();
		const buttons = [...root.querySelectorAll('main button')].map((b) => b.textContent?.trim());
		expect(buttons).toEqual(['New report', 'New report']);
	});
});

describe('ListPage', () => {
	it('names the table and the filter group', () => {
		const root = mount();
		expect(root.querySelector('table caption')?.textContent).toBe('Reports');
		expect(root.querySelector('[role="group"][aria-label="Filters"] label')?.textContent).toBe('Search reports');
	});
});
