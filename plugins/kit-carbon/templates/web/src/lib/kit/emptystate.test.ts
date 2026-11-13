// EmptyState, rendered on the server inside the shell (__fixtures__/EmptyStateFixture.svelte) and
// audited with axe. Delete this file and the fixture with EmptyState.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit as auditDom, serverDom } from './__fixtures__/dom';
import EmptyStateFixture from './__fixtures__/EmptyStateFixture.svelte';

let dom: JSDOM;

async function mount(props: Record<string, unknown> = {}): Promise<HTMLElement> {
	dom = serverDom(EmptyStateFixture, props);
	return dom.window.document.body;
}

const audit = () => auditDom(dom);

afterEach(() => dom?.window.close());

describe('EmptyState fixture', () => {
	it('has no axe violations', async () => {
		await mount();
		expect(await audit()).toEqual([]);
	});
});

describe('EmptyState', () => {
	it('is a heading with a reason, so the page is never blank', async () => {
		const root = await mount();
		const h = root.querySelector('[data-kit="empty-state"] h2, [data-kit="empty-state"] h3');
		expect(h?.textContent).toContain('No reports yet');
	});
});
