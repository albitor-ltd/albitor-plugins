// Button in every variant, rendered on the server inside the shell
// (__fixtures__/ButtonFixture.svelte) and audited with axe. Delete this file and the fixture with
// Button.svelte if the app has no button.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit as auditDom, serverDom } from './__fixtures__/dom';
import ButtonFixture from './__fixtures__/ButtonFixture.svelte';

let dom: JSDOM;

async function mount(props: Record<string, unknown> = {}): Promise<HTMLElement> {
	dom = serverDom(ButtonFixture, props);
	return dom.window.document.body;
}

const audit = () => auditDom(dom);

afterEach(() => dom?.window.close());

describe('Button fixture', () => {
	it('has no axe violations', async () => {
		await mount();
		expect(await audit()).toEqual([]);
	});
});

describe('Button', () => {
	it('is type=button unless told otherwise, so it never submits a form by accident', async () => {
		const root = await mount();
		const byText = (t: string) =>
			[...root.querySelectorAll('form button')].find((b) => b.textContent?.includes(t))!;
		expect(byText('Send report').getAttribute('type')).toBe('submit');
		for (const t of ['Save draft', 'Reset', 'Delete report']) {
			expect(byText(t).getAttribute('type'), t).toBe('button');
		}
	});

	it('maps each variant onto its Carbon kind', async () => {
		const root = await mount();
		const cls = (t: string) =>
			[...root.querySelectorAll('form button')].find((b) => b.textContent?.includes(t))!.className;
		expect(cls('Send report')).toContain('bx--btn--primary');
		expect(cls('Save draft')).toContain('bx--btn--secondary');
		expect(cls('Reset')).toContain('bx--btn--ghost');
		expect(cls('Delete report')).toContain('bx--btn--danger');
	});
});
