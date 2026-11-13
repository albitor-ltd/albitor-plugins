// Table, rendered on the server inside the shell (__fixtures__/TableFixture.svelte) and audited
// with axe. Delete this file and the fixture with Table.svelte if the app has no table.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import TableFixture from './__fixtures__/TableFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(TableFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Table', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('has a caption and scoped headers', () => {
		const root = mount();
		const table = root.querySelector('table')!;
		expect(table.querySelector('caption')?.textContent).toContain('Recent reports');
		for (const th of table.querySelectorAll('thead th')) {
			expect(th.getAttribute('scope')).toBe('col');
		}
	});

	it('renders cell values as text, never as markup', () => {
		const root = mount({ rowName: '<b id="pwned">x</b>' });
		expect(root.querySelector('#pwned')).toBeNull();
		expect(root.querySelector('table')!.textContent).toContain('<b id="pwned">x</b>');
	});
});
