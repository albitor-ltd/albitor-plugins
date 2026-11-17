// FilterChip, rendered on the server inside the shell (__fixtures__/FilterChipFixture.svelte) and
// audited with axe. Delete this file and the fixture with FilterChip.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import FilterChipFixture from './__fixtures__/FilterChipFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(FilterChipFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('FilterChip', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is a toggle button whose state is aria-pressed, in a labelled group', () => {
		const root = mount();
		const group = root.querySelector('[role="group"]')!;
		expect(group.getAttribute('aria-label')).toBe('Filters');
		const chips = [...group.querySelectorAll('button[data-kit="filter-chip"]')];
		expect(chips.map((c) => [c.textContent?.trim(), c.getAttribute('aria-pressed')])).toEqual([
			['Open only', 'true'],
			['Mine', 'false']
		]);
	});

	it('draws a check on a pressed chip, so the state is not colour alone', () => {
		const [open, mine] = [...mount().querySelectorAll('[data-kit="filter-chip"]')];
		expect(open.querySelector('svg')).not.toBeNull();
		expect(mine.querySelector('svg')).toBeNull();
	});
});
