// Skeleton, rendered on the server inside the shell (__fixtures__/SkeletonFixture.svelte) and
// audited with axe. Delete this file and the fixture with Skeleton.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import SkeletonFixture from './__fixtures__/SkeletonFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(SkeletonFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Skeleton', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is hidden from assistive technology, so the region says it is loading instead', () => {
		const root = mount();
		const skeletons = [...root.querySelectorAll('[data-kit="skeleton"]')];
		expect(skeletons).toHaveLength(2);
		for (const s of skeletons) expect(s.getAttribute('aria-hidden')).toBe('true');
		expect(root.querySelector('[aria-busy="true"]')?.textContent).toContain('Loading reports');
	});

	it('takes its size from its props', () => {
		const first = mount().querySelector<HTMLElement>('[data-kit="skeleton"]')!;
		expect(first.getAttribute('style')).toContain('60%');
		expect(first.getAttribute('style')).toContain('1.25rem');
	});
});
