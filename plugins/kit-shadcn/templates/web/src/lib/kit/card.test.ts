// Card, rendered on the server inside the shell (__fixtures__/CardFixture.svelte) and audited with
// axe. Delete this file and the fixture with Card.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import CardFixture from './__fixtures__/CardFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(CardFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Card', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is a group named by its own heading when it has a title, not a landmark', () => {
		const root = mount();
		const section = root.querySelector('[role="group"][data-kit="card"]')!;
		expect(root.querySelector('section')).toBeNull();
		const heading = section.querySelector('h2')!;
		expect(section.getAttribute('aria-labelledby')).toBe(heading.id);
		expect(heading.textContent).toBe('Recent activity');
		expect(section.querySelector('a')?.textContent).toBe('View all');
	});

	it('is a plain container, not an unnamed group, without a title', () => {
		const plain = mount().querySelector('div[data-kit="card"]:not([role])')!;
		expect(plain.hasAttribute('aria-labelledby')).toBe(false);
		expect(plain.querySelector('h2, h3, h4')).toBeNull();
	});
});
