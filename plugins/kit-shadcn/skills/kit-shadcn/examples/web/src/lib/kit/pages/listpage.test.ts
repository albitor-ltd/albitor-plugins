// ListPage in each state (__fixtures__/ListPageFixture.svelte), rendered on the server the way a
// build prerenders it, loaded into a DOM and audited with axe. Delete this file and the fixture
// with ListPage.svelte if the app has no list page.
import { JSDOM } from 'jsdom';
import { render } from 'svelte/server';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Table from '../Table.svelte';
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
	['list, empty', { rows: [] }],
	['list, stacked rows', { narrow: 'stacked' }],
	['list, priority columns', { narrow: 'priority' }]
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

	it('shows a failure as an alert with a retry button, not only a toast', () => {
		const root = mount({ status: 'error' });
		const alert = root.querySelector('[role="alert"]');
		expect(alert?.textContent).toContain('Something went wrong');
		expect(alert?.querySelector('button')?.textContent?.trim()).toBe('Try again');
	});

	it('says what to do next when a list is empty', () => {
		const root = mount({ rows: [] });
		expect(root.querySelector('[data-kit="empty-state"]')?.textContent).toContain('Create the first report');
		expect(root.querySelector('[data-kit="empty-state"] button')?.textContent?.trim()).toBe('New report');
	});
});

describe('ListPage', () => {
	it('names the table and the filter group', () => {
		const root = mount();
		expect(root.querySelector('table caption')?.textContent).toBe('Reports');
		expect(root.querySelector('[role="group"][aria-label="Filters"] label')?.textContent).toBe('Search reports');
	});
});

describe('ListPage narrow mode', () => {
	it('scrolls by default, with no stacked labels or disclosure', () => {
		const root = mount();
		expect(root.querySelector('[data-narrow]')?.getAttribute('data-narrow')).toBe('scroll');
		expect(root.querySelector('details')).toBeNull();
	});

	it('stacked rows label every non-row-header cell with its column', () => {
		const root = mount({ narrow: 'stacked' });
		expect(root.querySelector('[data-narrow]')?.getAttribute('data-narrow')).toBe('stacked');
		expect(root.querySelector('tbody td')?.getAttribute('data-label')).toBe('Owner');
		expect(root.querySelector('details')).toBeNull();
	});

	it('priority columns move secondary columns into a Details cell with a per-row name', () => {
		const root = mount({ narrow: 'priority' });
		expect(root.querySelector('thead th.secondary')?.textContent).toBe('Owner');
		expect(root.querySelector('tbody td')?.classList.contains('secondary')).toBe(true);
		const cell = root.querySelector('tbody td.more');
		expect(root.querySelector('tbody th[scope="row"] details')).toBeNull();
		expect(cell?.querySelector('summary')?.textContent).toBe('Details for Quarterly summary');
		expect(cell?.querySelector('dt')?.textContent).toBe('Owner');
		expect(cell?.querySelector('dd')?.textContent).toBe('Finance');
	});

	it('restores table roles on stacked rows', () => {
		const root = mount({ narrow: 'stacked' });
		expect(root.querySelector('table')?.getAttribute('role')).toBe('table');
		expect(root.querySelector('tbody tr')?.getAttribute('role')).toBe('row');
		expect(root.querySelector('tbody th')?.getAttribute('role')).toBe('rowheader');
		expect(root.querySelector('tbody td')?.getAttribute('role')).toBe('cell');
		expect(root.querySelector('thead th')?.getAttribute('role')).toBe('columnheader');
	});

	it('never hides the first column, whatever its priority', () => {
		const { body } = render(Table, {
			props: {
				caption: 'Reports',
				narrow: 'priority',
				columns: [
					{ key: 'name', label: 'Name', priority: 'secondary' },
					{ key: 'owner', label: 'Owner', priority: 'secondary' }
				],
				rows: [{ name: 'Quarterly summary', owner: 'Finance' }]
			}
		});
		const doc = new JSDOM(`<body>${body}</body>`).window.document;
		expect(doc.querySelector('thead th')?.classList.contains('secondary')).toBe(false);
		expect(doc.querySelector('tbody th')?.textContent).toBe('Quarterly summary');
	});
});
