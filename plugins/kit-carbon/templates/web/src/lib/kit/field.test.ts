// Field, rendered on the server inside the shell (__fixtures__/FieldFixture.svelte) and audited
// with axe. Delete this file and the fixture with Field.svelte if the app has no form.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit as auditDom, serverDom } from './__fixtures__/dom';
import FieldFixture from './__fixtures__/FieldFixture.svelte';

let dom: JSDOM;

async function mount(props: Record<string, unknown> = {}): Promise<HTMLElement> {
	dom = serverDom(FieldFixture, props);
	return dom.window.document.body;
}

const audit = () => auditDom(dom);

afterEach(() => dom?.window.close());

describe('Field fixture', () => {
	it('has no axe violations', async () => {
		await mount();
		expect(await audit()).toEqual([]);
	});
});

describe('Field', () => {
	it('ties the label, hint and error to the control', async () => {
		const root = await mount();
		const input = root.querySelector<HTMLInputElement>('input[type="email"]')!;
		const label = root.querySelector(`label[for="${input.id}"]`);
		expect(label?.textContent).toContain('Email address');
		expect(input.getAttribute('aria-invalid')).toBe('true');
		const described = input.getAttribute('aria-describedby')!.split(' ');
		expect(described).toHaveLength(2);
		for (const id of described) expect(root.querySelector(`#${id}`)).not.toBeNull();
		expect(root.querySelector(`#${described[1]}`)!.textContent).toContain('Enter an email address');
	});

	it('keeps a caller\'s aria-describedby and adds the hint and error ids to it', async () => {
		const root = await mount();
		const notes = root.querySelector('textarea')!;
		expect(notes.getAttribute('aria-describedby')).toBe('notes-help');
		const email = root.querySelector('input[type="email"]')!;
		// The field's own wiring comes first and is never replaced by a caller's attribute.
		expect(email.getAttribute('aria-describedby')!.split(' ')).toHaveLength(2);
	});

	it('gives two fields different ids', async () => {
		const root = await mount();
		const ids = [...root.querySelectorAll('input, textarea, select')].map((e) => e.id);
		expect(ids.every(Boolean)).toBe(true);
		expect(new Set(ids).size).toBe(ids.length);
	});
});
