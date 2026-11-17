// StatTile, rendered on the server inside the shell (__fixtures__/StatTileFixture.svelte) and
// audited with axe. Delete this file and the fixture with StatTile.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import StatTileFixture from './__fixtures__/StatTileFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(StatTileFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('StatTile', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('reads the figure with its label', () => {
		const tile = mount().querySelector('[data-kit="stat-tile"]')!;
		expect(tile.querySelector('dt')?.textContent?.trim()).toBe('Open items');
		expect(tile.querySelector('dd.value')?.textContent).toBe('12');
	});

	it('says the change in words; the arrow is decoration', () => {
		const delta = mount().querySelector('.delta')!;
		expect(delta.textContent).toContain('3 more than last week');
		expect(delta.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true');
	});

	it('hides the icon from assistive technology, since the label is beside it', () => {
		const badge = mount().querySelector('.badge svg')!;
		expect(badge.getAttribute('aria-hidden')).toBe('true');
	});

	it('draws a meter as a named progressbar', () => {
		const bar = mount().querySelector('[data-kit="stat-tile"] [role="progressbar"]')!;
		expect(bar.getAttribute('aria-label')).toBe('Storage used');
		expect(bar.getAttribute('aria-valuenow')).toBe('62');
	});

	it('draws a sparkline with its values available as text', () => {
		const tiles = mount().querySelectorAll('[data-kit="stat-tile"]');
		const spark = tiles[2].querySelector('[data-kind="sparkline"]')!;
		expect(spark.querySelector('caption')?.textContent).toBe('Signups, last 3 days');
		expect(spark.querySelectorAll('tbody tr')).toHaveLength(3);
	});

	it('leaves out what it was not given', () => {
		const plain = mount().querySelectorAll('[data-kit="stat-tile"]')[1];
		expect(plain.querySelector('.delta')).toBeNull();
		expect(plain.querySelector('.badge')).toBeNull();
	});
});
