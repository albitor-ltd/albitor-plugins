// Progress (bar and ring), rendered on the server inside the shell
// (__fixtures__/ProgressFixture.svelte) and audited with axe. Delete this file and the fixture
// with Progress.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import ProgressFixture from './__fixtures__/ProgressFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(ProgressFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Progress', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is a named progressbar with its value, for the bar and the ring', () => {
		const bars = [...mount().querySelectorAll('[role="progressbar"]')];
		expect(bars.map((b) => b.getAttribute('aria-label'))).toEqual([
			'Storage used',
			'Setup steps',
			'Importing'
		]);
		expect(bars[0].getAttribute('aria-valuenow')).toBe('62');
		expect(bars[1].getAttribute('aria-valuenow')).toBe('3');
		expect(bars[1].getAttribute('aria-valuemax')).toBe('5');
	});

	it('is indeterminate with no value rather than showing zero', () => {
		const bar = mount().querySelectorAll('[role="progressbar"]')[2];
		expect(bar.hasAttribute('aria-valuenow')).toBe(false);
	});

	it('writes the figure as text too, so it is not only a length', () => {
		const root = mount();
		expect(root.textContent).toContain('62%');
		expect(root.textContent).toContain('60%');
	});

	it('draws the bar and the ring in proportion', () => {
		const root = mount();
		expect(root.querySelector<HTMLElement>('.bar-wrap .fill')!.getAttribute('style')).toContain('62%');
		const ring = root.querySelector('circle.fill')!;
		const circumference = Number(ring.getAttribute('stroke-dasharray'));
		expect(Number(ring.getAttribute('stroke-dashoffset'))).toBeCloseTo(circumference * 0.4, 3);
	});
});
