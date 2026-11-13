// DashboardPage in each state (__fixtures__/DashboardPageFixture.svelte), rendered on the server
// the way a build prerenders it, loaded into a DOM and audited with axe. Delete this file and the
// fixture with DashboardPage.svelte if the app has no dashboard.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import DashboardPageFixture from './__fixtures__/DashboardPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(DashboardPageFixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['dashboard', {}],
	['dashboard, loading', { status: 'loading' }],
	['dashboard, failed', { status: 'error' }]
];

describe('DashboardPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('DashboardPage', () => {
	it('reads each figure with its label', () => {
		const root = mount();
		const first = root.querySelector('.stat');
		expect(first?.querySelector('dt')?.textContent).toBe('Open items');
		expect(first?.querySelector('dd')?.textContent).toBe('12');
	});
});
