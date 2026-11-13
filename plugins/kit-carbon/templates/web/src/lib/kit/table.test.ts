// Table, rendered on the server inside the shell (__fixtures__/TableFixture.svelte) and audited
// with axe. Delete this file and the fixture with Table.svelte if the app has no table.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit as auditDom, serverDom } from './__fixtures__/dom';
import TableFixture from './__fixtures__/TableFixture.svelte';

let dom: JSDOM;

async function mount(props: Record<string, unknown> = {}): Promise<HTMLElement> {
	dom = serverDom(TableFixture, props);
	return dom.window.document.body;
}

const audit = () => auditDom(dom);

afterEach(() => dom?.window.close());

describe('Table fixture', () => {
	it('has no axe violations', async () => {
		await mount();
		expect(await audit()).toEqual([]);
	});
});

describe('Table', () => {
	it('has a caption and scoped headers', async () => {
		const root = await mount();
		const table = root.querySelector('table')!;
		expect(table.querySelector('caption')?.textContent).toContain('Recent reports');
		for (const th of table.querySelectorAll('thead th')) {
			expect(th.getAttribute('scope')).toBe('col');
		}
	});

	it('renders cell values as text, never as markup', async () => {
		const root = await mount({ rowName: '<b id="pwned">x</b>' });
		expect(root.querySelector('#pwned')).toBeNull();
		expect(root.querySelector('table')!.textContent).toContain('<b id="pwned">x</b>');
	});
});
