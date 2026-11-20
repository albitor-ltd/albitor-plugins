// EmptyState, rendered on the server inside the shell (__fixtures__/EmptyStateFixture.svelte) and
// audited with axe. Delete this file and the fixture with EmptyState.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import EmptyStateFixture from './__fixtures__/EmptyStateFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(EmptyStateFixture);
	return dom.window.document.body;
}

function mountWithArt(): HTMLElement {
	dom = serverDom(EmptyStateFixture, { art: true });
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('EmptyState', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is a heading with a reason, so the page is never blank', () => {
		const root = mount();
		const h = root.querySelector('[data-kit="empty-state"] h2, [data-kit="empty-state"] h3');
		expect(h?.textContent).toContain('No reports yet');
	});

	it('draws the look art above the title, decorative, in place of the icon', () => {
		const root = mountWithArt();
		const art = root.querySelector('[data-kit="empty-state"] .art-slot svg');
		expect(art?.getAttribute('aria-hidden')).toBe('true');
		expect(root.querySelector('[data-kit="empty-state"] .art')).toBeNull();
	});
});
