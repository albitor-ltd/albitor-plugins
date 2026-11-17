// Pill, rendered on the server inside the shell (__fixtures__/PillFixture.svelte) and audited with
// axe. Delete this file and the fixture with Pill.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import PillFixture from './__fixtures__/PillFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(PillFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Pill', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('never carries a status by colour alone: every tone draws a hidden glyph beside its word', () => {
		const pills = [...mount().querySelectorAll('[data-kit="pill"]')];
		expect(pills.map((p) => p.textContent?.trim())).toEqual([
			'Draft',
			'Live',
			'Needs review',
			'Failed',
			'New'
		]);
		for (const pill of pills) {
			const glyph = pill.querySelector('svg');
			expect(glyph, pill.textContent ?? '').not.toBeNull();
			expect(glyph!.getAttribute('aria-hidden')).toBe('true');
		}
	});

	it('uses a different glyph for each tone', () => {
		const shapes = [...mount().querySelectorAll('[data-kit="pill"] svg')].map((s) => s.innerHTML);
		expect(new Set(shapes).size).toBe(shapes.length);
	});
});
