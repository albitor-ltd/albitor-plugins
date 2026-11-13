// DetailPage in each state (__fixtures__/DetailPageFixture.svelte), rendered on the server the way a
// build prerenders it, loaded into a DOM and audited with axe. Delete this file and the fixture
// with DetailPage.svelte if the app has no such page.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import DetailPageFixture from './__fixtures__/DetailPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(DetailPageFixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['detail', {}],
	['detail, loading', { status: 'loading' }],
	['detail, failed', { status: 'error' }]
];

describe('DetailPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('DetailPage', () => {
	it('pairs every term with its value and links back to the list', () => {
		const root = mount();
		expect([...root.querySelectorAll('dt')].map((t) => t.textContent)).toEqual(['Owner', 'Status', 'Notes']);
		expect(root.querySelectorAll('dd')).toHaveLength(3);
		expect(root.querySelector('dl.govuk-summary-list')).not.toBeNull();
		expect(root.querySelector('main a[href="/reports"]')?.textContent).toBe('All reports');
	});

	it('says "Not set" for an empty value rather than leaving a blank', () => {
		const root = mount();
		expect(root.querySelectorAll('dd')[2]?.textContent).toBe('Not set');
	});
});
